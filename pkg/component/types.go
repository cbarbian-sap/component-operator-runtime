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

package component

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

// Component is the central interface that component operators have to implement.
// Besides being a conroller-runtime client.Object, the implmenting type has to include
// the Spec and Status structs defined in this package, and has to define according accessor methods,
// called GetComponentSpec() and GetComponentStatus(). In addition it has to expose its whole spec and status
// as Unstructurable objects, via methods GetSpec() and GetStatus().
type Component interface {
	client.Object
	// Return target namespace for the component deployment.
	// This is the value that will be passed to Generator.Generate() as namespace.
	// In addition, rendered namespaced resources without namespace will be placed in this namespace.
	GetDeploymentNamespace() string
	// Return target name for the component deployment.
	// This is the value that will be passed to Generator.Generator() as name.
	GetDeploymentName() string
	// Return a pointer accessor to the component's spec.
	// Which, as a consequence, obviously has to implement the types.Unstructurable interface.
	GetSpec() types.Unstructurable
	// Return a pointer accessor to the component's status,
	// resp. to the corresponding sub-struct if the status extends component.Status.
	GetStatus() *Status
}

// +kubebuilder:object:generate=true

// Component Spec. Types implementing the Component interface may include this into their spec.
type Spec struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// +kubebuilder:object:generate=true

// Component Status. Types implementing the Component interface must include this into their status.
type Status struct {
	ObservedGeneration int64        `json:"observedGeneration"`
	AppliedGeneration  int64        `json:"appliedGeneration,omitempty"`
	LastObservedAt     *metav1.Time `json:"lastObservedAt,omitempty"`
	LastAppliedAt      *metav1.Time `json:"lastAppliedAt,omitempty"`
	Conditions         []Condition  `json:"conditions,omitempty"`
	// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
	State     State            `json:"state,omitempty"`
	Inventory []*InventoryItem `json:"inventory,omitempty"`
}

// +kubebuilder:object:generate=true

// Component status Condition.
type Condition struct {
	Type ConditionType `json:"type"`
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status ConditionStatus `json:"status"`
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

// Condition type. Currently, only the 'Ready' type is used.
type ConditionType string

const (
	// Condition type representing the 'Ready' condition.
	ConditionTypeReady ConditionType = "Ready"
)

// Condition Status. Can be one of 'True', 'False', 'Unknown'.
type ConditionStatus string

const (
	// Condition status 'True'.
	ConditionTrue ConditionStatus = "True"
	// Condition status 'False'.
	ConditionFalse ConditionStatus = "False"
	// Condition status 'Unknown'.
	ConditionUnknown ConditionStatus = "Unknown"
)

// Component state. Can be one of 'Ready', 'Processing', 'Error', 'Deleting'.
type State string

const (
	// Component state 'Ready'.
	StateReady State = "Ready"
	// Component state 'Processing'.
	StateProcessing State = "Processing"
	// Component state 'Error'.
	StateError State = "Error"
	// Component state 'Deleting'.
	StateDeleting State = "Deleting"
)

// TypeInfo represents a Kubernetes type.
type TypeInfo struct {
	// API group.
	Group string `json:"group"`
	// API group version.
	Version string `json:"version"`
	// API kind.
	Kind string `json:"kind"`
}

// NameInfo represents an object's namespace and name.
type NameInfo struct {
	// Namespace of the referenced object; empty for non-namespaced objects
	Namespace string `json:"namespace,omitempty"`
	// Name of the referenced object.
	Name string `json:"name"`
}

// +kubebuilder:object:generate=true

// InventoryItem represents a dependent object managed by this operator.
type InventoryItem struct {
	// Type of the dependent object.
	TypeInfo `json:",inline"`
	// Namespace and name of the dependent object.
	NameInfo `json:",inline"`
	// Managed types
	ManagedTypes []TypeInfo `json:"managedTypes,omitempty"`
	// Digest of the descriptor of the dependent object.
	Digest string `json:"digest"`
	// Phase of the dependent object.
	Phase string `json:"phase,omitempty"`
	// Observed status of the dependent object, as observed by kstatus.
	Status string `json:"status,omitempty"`
}

const (
	PhaseScheduledForApplication = "ScheduledForApplication"
	PhaseScheduledForDeletion    = "ScheduledForDeletion"
	PhaseScheduledForCompletion  = "ScheduledForCompletion"
	PhaseCreating                = "Creating"
	PhaseUpdating                = "Updating"
	PhaseDeleting                = "Deleting"
	PhaseCompleting              = "Completing"
	PhaseReady                   = "Ready"
	PhaseCompleted               = "Completed"
)
