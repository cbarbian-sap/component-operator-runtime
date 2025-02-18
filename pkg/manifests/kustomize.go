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

package manifests

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/krusty"
	kustypes "sigs.k8s.io/kustomize/api/types"
	kustfsys "sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/sap/component-operator-runtime/internal/templatex"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// KustomizeGenerator is a Generator implementation that basically renders a given Kustomization.
type KustomizeGenerator struct {
	kustomizer *krusty.Kustomizer
	templates  []*template.Template
}

var _ Generator = &KustomizeGenerator{}

// Create a new KustomizeGenerator.
func NewKustomizeGenerator(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client) (*KustomizeGenerator, error) {
	g := KustomizeGenerator{}

	if fsys == nil {
		fsys = os.DirFS("/")
		absoluteKustomizationPath, err := filepath.Abs(kustomizationPath)
		if err != nil {
			return nil, err
		}
		kustomizationPath = absoluteKustomizationPath[1:]
	}

	options := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}
	g.kustomizer = krusty.MakeKustomizer(options)

	var t *template.Template
	if err := fs.WalkDir(
		fsys,
		kustomizationPath,
		func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !dirEntry.Type().IsRegular() {
				return nil
			}
			if !strings.HasSuffix(path, templateSuffix) {
				return nil
			}
			raw, err := fs.ReadFile(fsys, path)
			if err != nil {
				return err
			}
			name, err := filepath.Rel(kustomizationPath, path)
			if err != nil {
				// TODO: is it ok to panic here in case of error ?
				panic(err)
			}
			if t == nil {
				t = template.New(name)
			} else {
				t = t.New(name)
			}
			t.Option("missingkey=zero").
				Funcs(sprig.TxtFuncMap()).
				Funcs(templatex.FuncMap()).
				Funcs(templatex.FuncMapForTemplate(t)).
				Funcs(templatex.FuncMapForClient(client))
			if _, err := t.Parse(string(raw)); err != nil {
				return err
			}
			g.templates = append(g.templates, t)
			return nil
		},
	); err != nil {
		return nil, err
	}

	return &g, nil
}

// Create a new KustomizeGenerator with a ParameterTransformer attached (further transformers can be attached to the reeturned generator object).
func NewKustomizeGeneratorWithParameterTransformer(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client, transformer ParameterTransformer) (TransformableGenerator, error) {
	g, err := NewKustomizeGenerator(fsys, kustomizationPath, templateSuffix, client)
	if err != nil {
		return nil, err
	}
	return NewGenerator(g).WithParameterTransformer(transformer), nil
}

// Create a new KustomizeGenerator with an ObjectTransformer attached (further transformers can be attached to the reeturned generator object).
func NewKustomizeGeneratorWithObjectTransformer(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client, transformer ObjectTransformer) (TransformableGenerator, error) {
	g, err := NewKustomizeGenerator(fsys, kustomizationPath, templateSuffix, client)
	if err != nil {
		return nil, err
	}
	return NewGenerator(g).WithObjectTransformer(transformer), nil
}

// Generate resource descriptors.
func (g *KustomizeGenerator) Generate(namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	var objects []client.Object

	data := parameters.ToUnstructured()
	fsys := kustfsys.MakeFsInMemory()

	for _, t := range g.templates {
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return nil, err
		}
		if err := fsys.WriteFile(t.Name(), buf.Bytes()); err != nil {
			return nil, err
		}
	}

	resmap, err := g.kustomizer.Run(fsys, "/")
	if err != nil {
		return nil, err
	}

	raw, err := resmap.AsYaml()
	if err != nil {
		return nil, err
	}

	decoder := utilyaml.NewYAMLToJSONDecoder(bytes.NewBuffer(raw))
	for {
		object := &unstructured.Unstructured{}
		if err := decoder.Decode(&object.Object); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if object.Object == nil {
			continue
		}
		objects = append(objects, object)
	}

	return objects, nil
}
