/*
Copyright {{ now.Year }} {{ .owner }}.

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

package {{ .groupVersion }}

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/component"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"
)

// {{ .kind }}Spec defines the desired state of {{ .kind }}
type {{ .kind }}Spec struct {
	// You can remove component.Spec, but then you have to provide your own (meaningful)
	// implementations of GetDeploymentNamespace() and GetDeploymentName() below.
	component.Spec `json:",inline"`
	// Add your own fields here, describing the deployment of the managed component.
}

// {{ .kind }}Status defines the observed state of {{ .kind }}
type {{ .kind }}Status struct {
	component.Status `json:",inline"`
	// You may add your own fields here; this is rarely needed.
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// {{ .kind }} is the Schema for the {{ .resource }} API
type {{ .kind }} struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec {{ .kind }}Spec `json:"spec,omitempty"`
	// +kubebuilder:default={"observedGeneration":-1}
	Status {{ .kind }}Status `json:"status,omitempty"`
}

var _ component.Component = &{{ .kind }}{}

// +kubebuilder:object:root=true

// {{ .kind }}List contains a list of {{ .kind }}
type {{ .kind }}List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []{{ .kind }} `json:"items"`
}

func (s *{{ .kind }}Spec) ToUnstructured() map[string]any {
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(s)
	if err != nil {
		panic(err)
	}
	return result
}

func (c *{{ .kind }}) GetDeploymentNamespace() string {
	if c.Spec.Namespace != "" {
		return c.Spec.Namespace
	}
	return c.Namespace
}

func (c *{{ .kind }}) GetDeploymentName() string {
	if c.Spec.Name != "" {
		return c.Spec.Name
	}
	return c.Name
}

func (c *{{ .kind }}) GetSpec() componentoperatorruntimetypes.Unstructurable {
	return &c.Spec
}

func (c *{{ .kind }}) GetStatus() *component.Status {
	return &c.Status.Status
}

// The following post read hook ensures that Spec.Namespace and Spec.Name are properly defaulted (as metadata.namespace/metadata.name).
// The hook will be called by the reconciler after retrieving the component object from the Kubernetes API.
// Of course, the same could be (better) achieved by a mutating admission webhook, but we strive to build component operators
// without admission webhooks.
// You can remove this hook (and its registration in pkg/operator/operator.go) if you do not use .Spec.Namespace or .Spec.Name anywhere,
// or if you have chosen not to include component.Spec in {{.kind}}Spec above.
func PostReadHook(ctx context.Context, client client.Client, c *{{ .kind }}) error {
	if c.Spec.Namespace == "" {
		c.Spec.Namespace = c.Namespace
	}
	if c.Spec.Name == "" {
		c.Spec.Name = c.Name
	}
	return nil
}

func init() {
	SchemeBuilder.Register(&{{ .kind }}{}, &{{ .kind }}List{})
}
