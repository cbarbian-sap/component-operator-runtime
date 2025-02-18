---
title: "Helm Generator"
linkTitle: "Helm Generator"
weight: 20
type: "docs"
description: >
  A resource generator for helm charts
---

Sometimes it is desired to write a component operator (using component-operator-runtime) for some cluster component, which already has a productive Helm chart. Then it can make sense to use the `HelmGenerator` implementation of the `Generator` interface included in this module:

```go
package manifests

func NewHelmGenerator(
  name                  string,
  fsys                  fs.FS,
  chartPath             string,
  client                client.Client,
  discoveryClient       discovery.DiscoveryInterface
) (*HelmGenerator, error)
```

Here:
- `name` must be identical to the name used for the related `Reconciler`
- `fsys` must be an implementation of `fs.FS`, such as `embed.FS`; or it can be passed as nil; then, all file operations will be executed on the current OS filesystem.
- `chartPath` is the path containing the used Helm chart; if `fsys` was provided, this has to be a relative path; otherwise, it will be interpreted with respect to the OS filesystem (as an absolute path, or relative to the current working directory of the controller).
- `client` and `discoveryClient` should be the same ones used in the `Reconciler` consuming this generator.

It should be noted that `HelmGenerator` does not use the Helm SDK; instead it tries to emulate the Helm behavior as good as possible.
A few differences and restrictions arise from this:
- Subcharts are not supported
- Not all Helm template functions are supported. To be exact, `toToml`, `fromYamlArray`, `fromJsonArray` are not supported;
  the functions `toYaml`, `fromYaml`, `toJson`, `fromJson` are supported, but will behave more strictly in error situtations.
- Not all builtin variables are supported;
  for the `.Release` builtin, only `.Release.Namespace`, `.Release.Name`, `.Release.Service` are supported;
  for the `.Chart` builtin, only `.Chart.Name`, `.Chart.Version`, `.Chart.AppVersion` are supported;
  for the `.Capabilities` builtin, only `.Capabilities.KubeVersion` and `.Capabilities.APIVersions` are supported;
  the `.Template` builtin is fully supported; the `.Files` builtin is not supported at all.
- Regarding hooks, `pre-delete` and `post-delete` hooks are not allowed; test and rollback hooks are ignored, and `pre-install`,  `post-install`, `pre-upgrade`, `post-upgrade` hooks might be handled in a sligthly different way; hook weights will be handled in a compatible way; hook deletion policy `hook-failed` is not allowed, but `before-hook-creation` and `hook-succeeded` should work as expected.