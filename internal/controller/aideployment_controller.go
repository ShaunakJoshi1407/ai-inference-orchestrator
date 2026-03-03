package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"
)

type AIDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *AIDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var aiDeploy infrav1.AIDeployment
	if err := r.Get(ctx, req.NamespacedName, &aiDeploy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	labels := map[string]string{
		"app": aiDeploy.Name,
	}

	// -----------------------------------------------------
	// Resource Defaults
	// -----------------------------------------------------
	var containerResources corev1.ResourceRequirements

	if aiDeploy.Spec.Resources != nil {
		containerResources = *aiDeploy.Spec.Resources
	} else {
		containerResources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		}
	}

	// -----------------------------------------------------
	// Deployment (Conflict-Safe Production Pattern)
	// -----------------------------------------------------
	deploymentName := fmt.Sprintf("%s-deployment", aiDeploy.Name)

	var deployment appsv1.Deployment
	err := r.Get(ctx, client.ObjectKey{
		Name:      deploymentName,
		Namespace: aiDeploy.Namespace,
	}, &deployment)

	if apierrors.IsNotFound(err) {

		newDeployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: aiDeploy.Namespace,
				Labels:    labels,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "ollama-models",
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
								},
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:    "model-pull",
								Image:   "ollama/ollama:latest",
								Command: []string{"sh"},
								Args: []string{
									"-c",
									fmt.Sprintf("ollama serve & sleep 5 && ollama pull %s", aiDeploy.Spec.Model),
								},
								Env: []corev1.EnvVar{
									{Name: "OLLAMA_HOST", Value: "0.0.0.0"},
									{Name: "OLLAMA_MODELS", Value: "/models"},
								},
								VolumeMounts: []corev1.VolumeMount{
									{Name: "ollama-models", MountPath: "/models"},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:    "ollama",
								Image:   "ollama/ollama:latest",
								Command: []string{"ollama"},
								Args:    []string{"serve"},
								Ports: []corev1.ContainerPort{
									{ContainerPort: 11434},
								},
								Env: []corev1.EnvVar{
									{Name: "OLLAMA_HOST", Value: "0.0.0.0"},
									{Name: "OLLAMA_MODELS", Value: "/models"},
								},
								Resources: containerResources,
								VolumeMounts: []corev1.VolumeMount{
									{Name: "ollama-models", MountPath: "/models"},
								},
							},
						},
					},
				},
			},
		}

		if aiDeploy.Spec.Autoscaling == nil || !aiDeploy.Spec.Autoscaling.Enabled {
			replicas := int32(1)
			if aiDeploy.Spec.Replicas != nil {
				replicas = *aiDeploy.Spec.Replicas
			}
			newDeployment.Spec.Replicas = &replicas
		}

		if err := ctrl.SetControllerReference(&aiDeploy, &newDeployment, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &newDeployment); err != nil {
			return ctrl.Result{}, err
		}

	} else if err != nil {
		return ctrl.Result{}, err

	} else {

		original := deployment.DeepCopy()

		deployment.Labels = labels
		deployment.Spec.Template.Labels = labels

		if aiDeploy.Spec.Autoscaling == nil || !aiDeploy.Spec.Autoscaling.Enabled {
			replicas := int32(1)
			if aiDeploy.Spec.Replicas != nil {
				replicas = *aiDeploy.Spec.Replicas
			}
			deployment.Spec.Replicas = &replicas
		}

		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "ollama-models",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}

		deployment.Spec.Template.Spec.InitContainers = []corev1.Container{
			{
				Name:    "model-pull",
				Image:   "ollama/ollama:latest",
				Command: []string{"sh"},
				Args: []string{
					"-c",
					fmt.Sprintf("ollama serve & sleep 5 && ollama pull %s", aiDeploy.Spec.Model),
				},
				Env: []corev1.EnvVar{
					{Name: "OLLAMA_HOST", Value: "0.0.0.0"},
					{Name: "OLLAMA_MODELS", Value: "/models"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "ollama-models", MountPath: "/models"},
				},
			},
		}

		deployment.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:    "ollama",
				Image:   "ollama/ollama:latest",
				Command: []string{"ollama"},
				Args:    []string{"serve"},
				Ports: []corev1.ContainerPort{
					{ContainerPort: 11434},
				},
				Env: []corev1.EnvVar{
					{Name: "OLLAMA_HOST", Value: "0.0.0.0"},
					{Name: "OLLAMA_MODELS", Value: "/models"},
				},
				Resources: containerResources,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "ollama-models", MountPath: "/models"},
				},
			},
		}

		if err := r.Patch(ctx, &deployment, client.MergeFrom(original)); err != nil {
			logger.Error(err, "Failed to patch Deployment")
			return ctrl.Result{}, err
		}
	}

	// -----------------------------------------------------
	// Service (Patch Pattern)
	// -----------------------------------------------------
	serviceName := fmt.Sprintf("%s-service", aiDeploy.Name)

	var service corev1.Service
	err = r.Get(ctx, client.ObjectKey{
		Name:      serviceName,
		Namespace: aiDeploy.Namespace,
	}, &service)

	if apierrors.IsNotFound(err) {

		newService := corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: aiDeploy.Namespace,
				Labels:    labels,
			},
			Spec: corev1.ServiceSpec{
				Type:     corev1.ServiceTypeClusterIP,
				Selector: labels,
				Ports: []corev1.ServicePort{
					{
						Port:       8080,
						TargetPort: intstr.FromInt(11434),
					},
				},
			},
		}

		if err := ctrl.SetControllerReference(&aiDeploy, &newService, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &newService); err != nil {
			return ctrl.Result{}, err
		}

	} else if err != nil {
		return ctrl.Result{}, err
	}

	// -----------------------------------------------------
	// HPA
	// -----------------------------------------------------
	if aiDeploy.Spec.Autoscaling != nil && aiDeploy.Spec.Autoscaling.Enabled {

		hpaName := fmt.Sprintf("%s-hpa", aiDeploy.Name)

		var hpa autoscalingv2.HorizontalPodAutoscaler
		err := r.Get(ctx, client.ObjectKey{
			Name:      hpaName,
			Namespace: aiDeploy.Namespace,
		}, &hpa)

		if apierrors.IsNotFound(err) {

			newHPA := autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      hpaName,
					Namespace: aiDeploy.Namespace,
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       deploymentName,
					},
					MinReplicas: aiDeploy.Spec.Autoscaling.MinReplicas,
					MaxReplicas: aiDeploy.Spec.Autoscaling.MaxReplicas,
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: corev1.ResourceCPU,
								Target: autoscalingv2.MetricTarget{
									Type:               autoscalingv2.UtilizationMetricType,
									AverageUtilization: &aiDeploy.Spec.Autoscaling.TargetCPUUtilization,
								},
							},
						},
					},
				},
			}

			if err := ctrl.SetControllerReference(&aiDeploy, &newHPA, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, &newHPA); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *AIDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.AIDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Complete(r)
}
