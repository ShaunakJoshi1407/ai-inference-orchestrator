/*
Copyright 2026.
Licensed under the Apache License, Version 2.0 (the "License");
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AutoscalingSpec struct {
	Enabled bool `json:"enabled,omitempty"`

	MinReplicas *int32 `json:"minReplicas,omitempty"`

	MaxReplicas int32 `json:"maxReplicas,omitempty"`

	TargetCPUUtilization int32 `json:"targetCPUUtilization,omitempty"`
}

// AIDeploymentSpec defines the desired state of AIDeployment
type AIDeploymentSpec struct {

	// +kubebuilder:validation:Minimum=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:MinLength=1
	Model string `json:"model"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *int32 `json:"port,omitempty"`

	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +optional
	ServiceType *corev1.ServiceType `json:"serviceType,omitempty"`

	// +optional
	Image *string `json:"image,omitempty"`

	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	Autoscaling *AutoscalingSpec `json:"autoscaling,omitempty"`
}

// AIDeploymentStatus defines observed state
type AIDeploymentStatus struct {
	Replicas          int32 `json:"replicas,omitempty"`
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type AIDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AIDeploymentSpec   `json:"spec"`
	Status AIDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AIDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AIDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AIDeployment{}, &AIDeploymentList{})
}
