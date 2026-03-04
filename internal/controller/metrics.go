package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// Reconciliation metrics
	AIDeploymentReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aideployment_reconcile_total",
			Help: "Total number of AIDeployment reconciliations",
		},
		[]string{"name", "namespace"},
	)

	AIDeploymentReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aideployment_reconcile_errors_total",
			Help: "Total number of AIDeployment reconciliation errors",
		},
		[]string{"name", "namespace"},
	)

	AIDeploymentReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aideployment_reconcile_duration_seconds",
			Help:    "Reconciliation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"name", "namespace"},
	)

	// Business metrics
	AIDeploymentActiveTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "aideployment_active_total",
			Help: "Total number of active AIDeployments",
		},
	)

	AIDeploymentReplicasCurrent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aideployment_replicas_current",
			Help: "Current replica count per AIDeployment",
		},
		[]string{"name", "namespace"},
	)

	AIDeploymentReplicasAvailable = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aideployment_replicas_available",
			Help: "Available replica count per AIDeployment",
		},
		[]string{"name", "namespace"},
	)

	AIDeploymentAutoscalingEnabled = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aideployment_autoscaling_enabled",
			Help: "Whether autoscaling is enabled (1=true, 0=false)",
		},
		[]string{"name", "namespace"},
	)
)

func init() {
	ctrlmetrics.Registry.MustRegister(
		AIDeploymentReconcileTotal,
		AIDeploymentReconcileErrors,
		AIDeploymentReconcileDuration,
		AIDeploymentActiveTotal,
		AIDeploymentReplicasCurrent,
		AIDeploymentReplicasAvailable,
		AIDeploymentAutoscalingEnabled,
	)
}
