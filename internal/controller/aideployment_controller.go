package controller

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"
)

type AIDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// RBAC
// +kubebuilder:rbac:groups=infra.example.com,resources=aideployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.example.com,resources=aideployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infra.example.com,resources=aideployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *AIDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling AIDeployment", "name", req.NamespacedName)

	var aiDeploy infrav1.AIDeployment
	if err := r.Get(ctx, req.NamespacedName, &aiDeploy); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	replicas := int32(1)
	if aiDeploy.Spec.Replicas != nil {
		replicas = *aiDeploy.Spec.Replicas
	}

	port := int32(8000)
	if aiDeploy.Spec.Port != nil {
		port = *aiDeploy.Spec.Port
	}

	image := resolveModelImage(aiDeploy.Spec.Model)
	deploymentName := aiDeploy.Name + "-deployment"

	var existing appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{
		Name:      deploymentName,
		Namespace: aiDeploy.Namespace,
	}, &existing)

	if err != nil && errors.IsNotFound(err) {

		deployment := buildDeployment(aiDeploy, deploymentName, image, replicas, port)

		if err := ctrl.SetControllerReference(&aiDeploy, &deployment, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &deployment); err != nil {
			return ctrl.Result{}, err
		}

		meta.SetStatusCondition(&aiDeploy.Status.Conditions, metav1.Condition{
			Type:               "Progressing",
			Status:             metav1.ConditionTrue,
			Reason:             "Creating",
			Message:            "Deployment is being created",
			LastTransitionTime: metav1.Now(),
		})

		_ = r.Status().Update(ctx, &aiDeploy)

		return ctrl.Result{Requeue: true}, nil
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	// Drift detection
	desired := buildDeployment(aiDeploy, deploymentName, image, replicas, port)

	if !reflect.DeepEqual(existing.Spec.Replicas, desired.Spec.Replicas) ||
		existing.Spec.Template.Spec.Containers[0].Image != desired.Spec.Template.Spec.Containers[0].Image {

		log.Info("Updating Deployment to match desired state")

		existing.Spec.Replicas = desired.Spec.Replicas
		existing.Spec.Template.Spec.Containers[0].Image =
			desired.Spec.Template.Spec.Containers[0].Image

		if err := r.Update(ctx, &existing); err != nil {
			return ctrl.Result{}, err
		}

		meta.SetStatusCondition(&aiDeploy.Status.Conditions, metav1.Condition{
			Type:               "Progressing",
			Status:             metav1.ConditionTrue,
			Reason:             "Updating",
			Message:            "Deployment is updating",
			LastTransitionTime: metav1.Now(),
		})

		_ = r.Status().Update(ctx, &aiDeploy)

		return ctrl.Result{Requeue: true}, nil
	}

	// Production-grade status mapping
	for _, cond := range existing.Status.Conditions {

		switch cond.Type {

		case appsv1.DeploymentAvailable:
			meta.SetStatusCondition(&aiDeploy.Status.Conditions, metav1.Condition{
				Type:               "Available",
				Status:             metav1.ConditionStatus(cond.Status),
				Reason:             cond.Reason,
				Message:            cond.Message,
				LastTransitionTime: metav1.Now(),
			})

		case appsv1.DeploymentProgressing:
			meta.SetStatusCondition(&aiDeploy.Status.Conditions, metav1.Condition{
				Type:               "Progressing",
				Status:             metav1.ConditionStatus(cond.Status),
				Reason:             cond.Reason,
				Message:            cond.Message,
				LastTransitionTime: metav1.Now(),
			})

		case appsv1.DeploymentReplicaFailure:
			meta.SetStatusCondition(&aiDeploy.Status.Conditions, metav1.Condition{
				Type:               "Degraded",
				Status:             metav1.ConditionStatus(cond.Status),
				Reason:             cond.Reason,
				Message:            cond.Message,
				LastTransitionTime: metav1.Now(),
			})
		}
	}

	if err := r.Status().Update(ctx, &aiDeploy); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func buildDeployment(aiDeploy infrav1.AIDeployment, name, image string, replicas, port int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: aiDeploy.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "model",
							Image: image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: port},
							},
						},
					},
				},
			},
		},
	}
}

func resolveModelImage(model string) string {
	switch model {
	case "llama3":
		return "nginx"
	case "mistral":
		return "nginx"
	default:
		return "nginx"
	}
}

func (r *AIDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.AIDeployment{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
