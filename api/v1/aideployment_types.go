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
	// Enabled toggles HPA creation
	Enabled bool `json:"enabled,omitempty"`

	// Minimum number of replicas
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// Maximum number of replicas
	MaxReplicas int32 `json:"maxReplicas,omitempty"`

	// Target CPU utilization percentage
	TargetCPUUtilization int32 `json:"targetCPUUtilization,omitempty"`
}

// AIDeploymentSpec defines the desired state of AIDeployment
type AIDeploymentSpec struct {

	// Replicas is the number of model serving replicas
	// +kubebuilder:validation:Minimum=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Model is the model name to deploy (e.g. "llama3", "mistral")
	// +kubebuilder:validation:MinLength=1
	Model string `json:"model"`

	// Port is the container port exposed for inference
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *int32 `json:"port,omitempty"`

	// ServiceType controls how the service is exposed
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +optional
	ServiceType *corev1.ServiceType `json:"serviceType,omitempty"`

	// Image overrides the default resolved model image
	// +optional
	Image *string `json:"image,omitempty"`

	// Resources defines CPU/Memory requests and limits
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Autoscaling configuration
	// +optional
	Autoscaling *AutoscalingSpec `json:"autoscaling,omitempty"`
}

// AIDeploymentStatus defines the observed state of AIDeployment.
type AIDeploymentStatus struct {

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
