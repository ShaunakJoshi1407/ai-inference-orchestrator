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
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"
)

const defaultModelImage = "nginx"

type AIDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// RBAC
// +kubebuilder:rbac:groups=infra.example.com,resources=aideployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.example.com,resources=aideployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infra.example.com,resources=aideployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *AIDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := ctrl.LoggerFrom(ctx)

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
	if aiDeploy.Spec.Image != nil {
		image = *aiDeploy.Spec.Image
	}

	serviceType := corev1.ServiceTypeClusterIP
	if aiDeploy.Spec.ServiceType != nil {
		serviceType = *aiDeploy.Spec.ServiceType
	}

	deploymentName := aiDeploy.Name + "-deployment"
	serviceName := aiDeploy.Name + "-service"

	// --------------------
	// Deployment Reconcile
	// --------------------

	var existingDeploy appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{
		Name:      deploymentName,
		Namespace: aiDeploy.Namespace,
	}, &existingDeploy)

	desiredDeploy := buildDeployment(aiDeploy, deploymentName, image, replicas, port)

	if err != nil && errors.IsNotFound(err) {

		if err := ctrl.SetControllerReference(&aiDeploy, &desiredDeploy, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &desiredDeploy); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(existingDeploy.Spec.Replicas, desiredDeploy.Spec.Replicas) ||
		existingDeploy.Spec.Template.Spec.Containers[0].Image != desiredDeploy.Spec.Template.Spec.Containers[0].Image {

		existingDeploy.Spec = desiredDeploy.Spec

		if err := r.Update(ctx, &existingDeploy); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// --------------------
	// Service Reconcile
	// --------------------

	var existingSvc corev1.Service
	err = r.Get(ctx, types.NamespacedName{
		Name:      serviceName,
		Namespace: aiDeploy.Namespace,
	}, &existingSvc)

	desiredSvc := buildService(aiDeploy, serviceName, deploymentName, port, serviceType)

	if err != nil && errors.IsNotFound(err) {

		if err := ctrl.SetControllerReference(&aiDeploy, &desiredSvc, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &desiredSvc); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	if existingSvc.Spec.Type != desiredSvc.Spec.Type ||
		!reflect.DeepEqual(existingSvc.Spec.Ports, desiredSvc.Spec.Ports) {

		existingSvc.Spec.Type = desiredSvc.Spec.Type
		existingSvc.Spec.Ports = desiredSvc.Spec.Ports

		if err := r.Update(ctx, &existingSvc); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// --------------------
	// Status Mapping
	// --------------------

	for _, cond := range existingDeploy.Status.Conditions {

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
		}
	}

	if err := r.Status().Update(ctx, &aiDeploy); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled AIDeployment")

	return ctrl.Result{}, nil
}

func buildDeployment(aiDeploy infrav1.AIDeployment, name, image string, replicas, port int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: aiDeploy.Namespace,
			Labels: map[string]string{
				"app": name,
			},
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

func buildService(aiDeploy infrav1.AIDeployment, name, deploymentName string, port int32, svcType corev1.ServiceType) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: aiDeploy.Namespace,
			Labels: map[string]string{
				"app": deploymentName,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: svcType,
			Selector: map[string]string{
				"app": deploymentName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					TargetPort: intstrFromInt32(port),
				},
			},
		},
	}
}

func resolveModelImage(model string) string {
	switch model {
	case "llama3":
		return "ollama/ollama"
	case "mistral":
		return "ghcr.io/mistralai/mistral"
	default:
		return defaultModelImage
	}
}

func intstrFromInt32(i int32) intstr.IntOrString {
	return intstr.FromInt(int(i))
}

func (r *AIDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.AIDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
