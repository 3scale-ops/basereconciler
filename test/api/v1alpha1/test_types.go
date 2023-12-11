/*
Copyright 2021.

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

// Package v1alpha1 is a test API definition
// +kubebuilder:object:generate=true
// +groupName=example.com
package v1alpha1

import (
	"github.com/3scale-ops/basereconciler/reconciler"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "example.com", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// TestSpec defines the desired state of Test
type TestSpec struct {
	// +optional
	PDB *bool `json:"pdb,omitempty"`
	// +optional
	HPA *bool `json:"hpa,omitempty"`
	// +optional
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`
	// +optional
	PruneService *bool `json:"pruneService,omitempty"`
}

// TestStatus defines the observed state of Test
type TestStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	*appsv1.DeploymentStatus                  `json:"deploymentStatus,omitempty"`
	reconciler.UnimplementedStatefulSetStatus `json:"-"`
}

func (dss *TestStatus) GetDeploymentStatus(key types.NamespacedName) *appsv1.DeploymentStatus {
	return dss.DeploymentStatus
}

func (dss *TestStatus) SetDeploymentStatus(key types.NamespacedName, s *appsv1.DeploymentStatus) {
	dss.DeploymentStatus = s
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Test is the Schema for the tests API
type Test struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestSpec   `json:"spec,omitempty"`
	Status TestStatus `json:"status,omitempty"`
}

var _ reconciler.ObjectWithAppStatus = &Test{}

func (t *Test) GetStatus() reconciler.AppStatus {
	return &t.Status
}

// +kubebuilder:object:root=true

// TestList contains a list of Test
type TestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Test `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Test{}, &TestList{})
}
