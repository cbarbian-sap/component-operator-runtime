/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Config struct {
	Owner                    string `json:"owner,omitempty"`
	GroupName                string `json:"groupName,omitempty"`
	GroupVersion             string `json:"groupVersion,omitempty"`
	Kind                     string `json:"kind,omitempty"`
	Resource                 string `json:"resource,omitempty"`
	OperatorName             string `json:"operatorName,omitempty"`
	GoVersion                string `json:"goVersion,omitempty"`
	GoModule                 string `json:"goModule,omitempty"`
	Version                  string `json:"version,omitempty"`
	KubernetesVersion        string `json:"kubernetesVersion,omitempty"`
	ControllerRuntimeVersion string `json:"controllerRuntimeVersion,omitempty"`
	Image                    string `json:"image,omitempty"`
}

func (c Config) StringMap() map[string]any {
	raw, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		panic(err)
	}
	return result
}

type FS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

//go:embed all:templates
var templates embed.FS

// default verions
var (
	goVersion                = "1.19"
	version                  = "latest"
	kubernetesVersion        = "v0.26.1"
	controllerRuntimeVersion = "v0.14.2"
)

func main() {
	errlog := log.New(os.Stderr, "", 0)

	showVersion := false
	config := Config{Version: version}
	outputDir := ""

	pflag.BoolVar(&showVersion, "version", false, "Show version")
	pflag.StringVar(&config.Owner, "owner", "SAP SE", "Owner of this project, as written to the license header")
	pflag.StringVar(&config.GroupName, "group-name", "operator.kyma-project.io", "API group name")
	pflag.StringVar(&config.GroupVersion, "group-version", "v1alpha1", "API group version")
	pflag.StringVar(&config.Kind, "kind", "", "API kind for the component")
	pflag.StringVar(&config.Resource, "resource", "", "API resource (plural) for the component; if empty, it will be the pluralized kind")
	pflag.StringVar(&config.OperatorName, "operator-name", "", "Unique name for this operator, used e.g. for leader election and labels; should be a valid DNS hostname")
	pflag.StringVar(&config.GoVersion, "go-version", goVersion, "Go version to be used")
	pflag.StringVar(&config.GoModule, "go-module", "", "Name of the Go module, as written to the go.mod file")
	pflag.StringVar(&config.KubernetesVersion, "kubernetes-version", kubernetesVersion, "Kubernetes go-client version to be used")
	pflag.StringVar(&config.ControllerRuntimeVersion, "controller-runtime-version", controllerRuntimeVersion, "Controller-runtime version to be used")
	pflag.StringVar(&config.Image, "image", "controller:latest", "Name of the Docker/OCI image produced by this project")
	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		errlog.Printf("Usage: %s [options] [output directory]\n", filepath.Base(os.Args[0]))
		errlog.Printf("  [output directory]: Target directory for the generated scaffold; must exist\n")
		errlog.Printf("  [options]:\n")
		pflag.PrintDefaults()
	}
	pflag.Parse()
	outputDir = pflag.Arg(0)

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if outputDir == "" {
		errlog.Fatalf("no output directory provided")
	}
	if err := checkDirectoryExists(outputDir); err != nil {
		errlog.Fatal(err)
	}

	if err := validateAndDefaultConfig(&config); err != nil {
		errlog.Fatal(err)
	}

	if err := processTemplates(subFS(templates, "templates"), &config, outputDir); err != nil {
		errlog.Fatal(err)
	}

	// TODO: beautify the following a bit (e.g. only print the stdout/stderr in case of errors)
	fmt.Println(">>> Post-processing: go get")
	if err := run(outputDir, "go", "get"); err != nil {
		errlog.Fatal(err)
	}
	fmt.Println(">>> Post-processing: make generate")
	if err := run(outputDir, "make", "generate"); err != nil {
		errlog.Fatal(err)
	}
	fmt.Println(">>> Post-processing: make manifests")
	if err := run(outputDir, "make", "manifests"); err != nil {
		errlog.Fatal(err)
	}
	fmt.Println(">>> Post-processing: make fmt")
	if err := run(outputDir, "make", "fmt"); err != nil {
		errlog.Fatal(err)
	}
	fmt.Println(">>> Post-processing: make vet")
	if err := run(outputDir, "make", "vet"); err != nil {
		errlog.Fatal(err)
	}
}

func validateAndDefaultConfig(config *Config) error {
	if config.Owner == "" {
		return fmt.Errorf("missing or empty config flag: --owner")
	}

	if config.GroupName == "" {
		return fmt.Errorf("missing or empty config flag: --group-name")
	}
	// TODO: validate GroupName (is a valid DNS name)

	if config.GroupVersion == "" {
		return fmt.Errorf("missing or empty config flag: --group-version")
	}
	// TODO: validate GroupVersion is valid (what exactly does this mean?)

	if config.Kind == "" {
		return fmt.Errorf("missing or empty config flag: --kind")
	}
	// TODO: validate Kind is valid (camel case, only lettters?)

	if config.Resource == "" {
		gvr, _ := meta.UnsafeGuessKindToResource(schema.GroupVersionKind{Group: config.GroupName, Version: config.GroupName, Kind: config.Kind})
		config.Resource = gvr.Resource
	}
	// TODO: validate Resource (lower case, only letters?)

	if config.OperatorName == "" {
		return fmt.Errorf("missing or empty config flag: --operator-name")
	}
	// TODO: validate OperatorName (is a valid DNS name)

	if config.GoVersion == "" {
		return fmt.Errorf("missing or empty config flag: --go-version")
	}
	// TODO: validate GoVersion (major.minor)

	if config.GoModule == "" {
		return fmt.Errorf("missing or empty config flag: --go-module")
	}
	// TODO: validate GoModule

	if config.KubernetesVersion == "" {
		return fmt.Errorf("missing or empty config flag: --kubernetes-version")
	}
	// TODO: validate KubernetesVersion (vmajor.minor.patch)

	if config.ControllerRuntimeVersion == "" {
		return fmt.Errorf("missing or empty config flag: --controller-runtime-version")
	}
	// TODO: validate ControllerRuntimeVersion (vmajor.minor.patch)

	if config.Image == "" {
		return fmt.Errorf("missing or empty config flag: --image")
	}
	// TODO: validate Image

	return nil
}

func processTemplates(fsys FS, config *Config, outputDir string) error {
	entries, err := fsys.ReadDir(".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := entry.Name()
		outputPath := outputDir + "/" + path
		fsinfo, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if err := os.Mkdir(outputPath, 0755); err != nil {
				return err
			}
			if err := processTemplates(subFS(fsys, path), config, outputPath); err != nil {
				return err
			}
		} else {
			var output []byte
			if filepath.Ext(path) == ".tpl" {
				tpl, err := fsys.ReadFile(path)
				if err != nil {
					return err
				}
				output, err = renderTemplate(path, tpl, config.StringMap())
				if err != nil {
					return err
				}
				outputPath = strings.TrimSuffix(outputPath, ".tpl")
			} else {
				output, err = fsys.ReadFile(path)
				if err != nil {
					return err
				}
			}
			if err := os.WriteFile(outputPath, output, (fsinfo.Mode().Perm()|0600)&0755); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderTemplate(name string, tpl []byte, data any) ([]byte, error) {
	tmpl, err := template.New(name).Funcs(sprig.TxtFuncMap()).Option("missingkey=error").Parse(string(tpl))
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func subFS(fsys FS, dir string) FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub.(FS)
}

func checkDirectoryExists(path string) error {
	fsinfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !fsinfo.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}
	return nil
}

func run(pwd string, cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	command.Dir = pwd
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
