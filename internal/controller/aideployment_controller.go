package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
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

	// -----------------------------------------------------
	// Metrics: Reconcile Start
	// -----------------------------------------------------

	start := time.Now()

	AIDeploymentReconcileTotal.
		WithLabelValues(req.Name, req.Namespace).
		Inc()

	defer func() {
		duration := time.Since(start).Seconds()
		AIDeploymentReconcileDuration.
			WithLabelValues(req.Name, req.Namespace).
			Observe(duration)
	}()

	// -----------------------------------------------------

	var aiDeploy infrav1.AIDeployment
	if err := r.Get(ctx, req.NamespacedName, &aiDeploy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update active deployments metric
	var list infrav1.AIDeploymentList
	if err := r.List(ctx, &list); err == nil {
		AIDeploymentActiveTotal.Set(float64(len(list.Items)))
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
	// Deployment (Drift-aware + HPA-safe)
	// -----------------------------------------------------

	deploymentName := fmt.Sprintf("%s-deployment", aiDeploy.Name)

	var deployment appsv1.Deployment
	err := r.Get(ctx, client.ObjectKey{
		Name:      deploymentName,
		Namespace: aiDeploy.Namespace,
	}, &deployment)

	if apierrors.IsNotFound(err) {

		newDeployment := r.buildDeployment(&aiDeploy, containerResources, labels)

		if err := ctrl.SetControllerReference(&aiDeploy, newDeployment, r.Scheme); err != nil {
			AIDeploymentReconcileErrors.WithLabelValues(req.Name, req.Namespace).Inc()
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, newDeployment); err != nil {
			AIDeploymentReconcileErrors.WithLabelValues(req.Name, req.Namespace).Inc()
			return ctrl.Result{}, err
		}

	} else if err != nil {
		AIDeploymentReconcileErrors.WithLabelValues(req.Name, req.Namespace).Inc()
		return ctrl.Result{}, err
	} else {

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {

			var latest appsv1.Deployment
			if err := r.Get(ctx, client.ObjectKey{
				Name:      deploymentName,
				Namespace: aiDeploy.Namespace,
			}, &latest); err != nil {
				return err
			}

			desired := r.buildDeployment(&aiDeploy, containerResources, labels)
			desired.Spec.Selector = latest.Spec.Selector

			if aiDeploy.Spec.Autoscaling != nil && aiDeploy.Spec.Autoscaling.Enabled {
				desired.Spec.Replicas = latest.Spec.Replicas
			}

			if deploymentsEqual(&latest, desired) {
				return nil
			}

			latest.Spec = desired.Spec
			latest.Labels = desired.Labels

			return r.Update(ctx, &latest)
		})

		if err != nil {
			logger.Error(err, "Failed to reconcile Deployment")
			AIDeploymentReconcileErrors.WithLabelValues(req.Name, req.Namespace).Inc()
			return ctrl.Result{}, err
		}
	}

	// -----------------------------------------------------
	// Service
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
			AIDeploymentReconcileErrors.WithLabelValues(req.Name, req.Namespace).Inc()
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &newService); err != nil {
			AIDeploymentReconcileErrors.WithLabelValues(req.Name, req.Namespace).Inc()
			return ctrl.Result{}, err
		}
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

		targetCPU := aiDeploy.Spec.Autoscaling.TargetCPUUtilization
		if targetCPU == 0 {
			targetCPU = 60
		}

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
									AverageUtilization: &targetCPU,
								},
							},
						},
					},
					Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
						ScaleUp: &autoscalingv2.HPAScalingRules{
							StabilizationWindowSeconds: ptrInt32(30),
						},
						ScaleDown: &autoscalingv2.HPAScalingRules{
							StabilizationWindowSeconds: ptrInt32(120),
						},
					},
				},
			}

			if err := ctrl.SetControllerReference(&aiDeploy, &newHPA, r.Scheme); err != nil {
				AIDeploymentReconcileErrors.WithLabelValues(req.Name, req.Namespace).Inc()
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, &newHPA); err != nil {
				AIDeploymentReconcileErrors.WithLabelValues(req.Name, req.Namespace).Inc()
				return ctrl.Result{}, err
			}
		}
	}

	// -----------------------------------------------------
	// STATUS UPDATE
	// -----------------------------------------------------

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {

		var latestCR infrav1.AIDeployment
		if err := r.Get(ctx, req.NamespacedName, &latestCR); err != nil {
			return err
		}

		var updatedDeployment appsv1.Deployment
		if err := r.Get(ctx, client.ObjectKey{
			Name:      deploymentName,
			Namespace: aiDeploy.Namespace,
		}, &updatedDeployment); err != nil {
			return err
		}

		newStatus := latestCR.Status
		newStatus.Replicas = updatedDeployment.Status.Replicas
		newStatus.AvailableReplicas = updatedDeployment.Status.AvailableReplicas

		ready := updatedDeployment.Status.AvailableReplicas > 0

		condition := metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "PodsNotReady",
			Message:            "Waiting for available replicas",
		}

		if ready {
			condition.Status = metav1.ConditionTrue
			condition.Reason = "DeploymentAvailable"
			condition.Message = "Deployment has available replicas"
		}

		meta.SetStatusCondition(&newStatus.Conditions, condition)

		if reflect.DeepEqual(latestCR.Status, newStatus) {
			return nil
		}

		latestCR.Status = newStatus

		// Update business metrics
		AIDeploymentReplicasCurrent.
			WithLabelValues(aiDeploy.Name, aiDeploy.Namespace).
			Set(float64(newStatus.Replicas))

		AIDeploymentReplicasAvailable.
			WithLabelValues(aiDeploy.Name, aiDeploy.Namespace).
			Set(float64(newStatus.AvailableReplicas))

		autoscaling := 0.0
		if aiDeploy.Spec.Autoscaling != nil && aiDeploy.Spec.Autoscaling.Enabled {
			autoscaling = 1.0
		}

		AIDeploymentAutoscalingEnabled.
			WithLabelValues(aiDeploy.Name, aiDeploy.Namespace).
			Set(autoscaling)

		return r.Status().Update(ctx, &latestCR)
	})

	if err != nil {
		logger.Error(err, "Failed to update status")
		AIDeploymentReconcileErrors.WithLabelValues(req.Name, req.Namespace).Inc()
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AIDeploymentReconciler) buildDeployment(
	aiDeploy *infrav1.AIDeployment,
	containerResources corev1.ResourceRequirements,
	labels map[string]string,
) *appsv1.Deployment {

	replicas := int32(1)
	if aiDeploy.Spec.Replicas != nil {
		replicas = *aiDeploy.Spec.Replicas
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-deployment", aiDeploy.Name),
			Namespace: aiDeploy.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      "ollama",
							Image:     "ollama/ollama:latest",
							Command:   []string{"ollama"},
							Args:      []string{"serve"},
							Resources: containerResources,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 11434},
							},
						},
					},
				},
			},
		},
	}

	if aiDeploy.Spec.Autoscaling != nil && aiDeploy.Spec.Autoscaling.Enabled {
		deployment.Spec.Replicas = nil
	}

	return deployment
}

func deploymentsEqual(current, desired *appsv1.Deployment) bool {

	if !reflect.DeepEqual(current.Spec.Template.Spec, desired.Spec.Template.Spec) {
		return false
	}

	if !reflect.DeepEqual(current.Labels, desired.Labels) {
		return false
	}

	return true
}

func ptrInt32(i int32) *int32 {
	return &i
}

func (r *AIDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.AIDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Complete(r)
}
