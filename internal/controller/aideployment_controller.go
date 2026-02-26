package controller

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"
)

type AIDeploymentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// RBAC
// +kubebuilder:rbac:groups=infra.example.com,resources=aideployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.example.com,resources=aideployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *AIDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

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

	port := int32(8080)
	if aiDeploy.Spec.Port != nil {
		port = *aiDeploy.Spec.Port
	}

	deploymentName := aiDeploy.Name + "-deployment"

	var existingDeployment appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{
		Name:      deploymentName,
		Namespace: aiDeploy.Namespace,
	}, &existingDeployment)

	if err != nil && errors.IsNotFound(err) {

		deployment := buildDeployment(&aiDeploy, replicas, port)

		if err := ctrl.SetControllerReference(&aiDeploy, deployment, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Event(&aiDeploy,
			corev1.EventTypeNormal,
			"DeploymentCreated",
			"Deployment created successfully")

		return ctrl.Result{}, nil
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	// -------- SAFE DRIFT UPDATE --------

	updated := false

	// Update replicas
	if *existingDeployment.Spec.Replicas != replicas {
		existingDeployment.Spec.Replicas = &replicas
		updated = true
	}

	container := &existingDeployment.Spec.Template.Spec.Containers[0]

	desiredImage := resolveModelImage(&aiDeploy)
	if container.Image != desiredImage {
		container.Image = desiredImage
		updated = true
	}

	// Update resources
	if aiDeploy.Spec.Resources != nil &&
		!reflect.DeepEqual(container.Resources, *aiDeploy.Spec.Resources) {

		container.Resources = *aiDeploy.Spec.Resources
		updated = true
	}

	if updated {
		if err := r.Update(ctx, &existingDeployment); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Event(&aiDeploy,
			corev1.EventTypeNormal,
			"DeploymentUpdated",
			"Deployment updated to match desired state")
	}

	// -------- SERVICE --------

	serviceName := aiDeploy.Name + "-service"

	var existingService corev1.Service
	err = r.Get(ctx, types.NamespacedName{
		Name:      serviceName,
		Namespace: aiDeploy.Namespace,
	}, &existingService)

	if err != nil && errors.IsNotFound(err) {

		service := buildService(&aiDeploy, port)

		if err := ctrl.SetControllerReference(&aiDeploy, service, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, service); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Event(&aiDeploy,
			corev1.EventTypeNormal,
			"ServiceCreated",
			"Service created successfully")
	}

	updateStatusFromDeployment(&aiDeploy, &existingDeployment)
	_ = r.Status().Update(ctx, &aiDeploy)

	return ctrl.Result{}, nil
}

func (r *AIDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("aideployment-controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.AIDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func buildDeployment(aiDeploy *infrav1.AIDeployment, replicas int32, port int32) *appsv1.Deployment {

	labels := map[string]string{
		"app": aiDeploy.Name,
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aiDeploy.Name + "-deployment",
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
							Name:  "model",
							Image: resolveModelImage(aiDeploy),
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

func buildService(aiDeploy *infrav1.AIDeployment, port int32) *corev1.Service {

	serviceType := corev1.ServiceTypeClusterIP
	if aiDeploy.Spec.ServiceType != nil {
		serviceType = *aiDeploy.Spec.ServiceType
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aiDeploy.Name + "-service",
			Namespace: aiDeploy.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app": aiDeploy.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					TargetPort: intstr.FromInt(int(port)),
				},
			},
		},
	}
}

func resolveModelImage(aiDeploy *infrav1.AIDeployment) string {
	if aiDeploy.Spec.Image != nil {
		return *aiDeploy.Spec.Image
	}
	return "nginx"
}

func updateStatusFromDeployment(aiDeploy *infrav1.AIDeployment, deploy *appsv1.Deployment) {

	aiDeploy.Status.Conditions = nil

	for _, cond := range deploy.Status.Conditions {
		aiDeploy.Status.Conditions = append(aiDeploy.Status.Conditions, metav1.Condition{
			Type:               string(cond.Type),
			Status:             metav1.ConditionStatus(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastTransitionTime: cond.LastTransitionTime,
			ObservedGeneration: aiDeploy.Generation,
		})
	}
}
