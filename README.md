# AI Inference Orchestrator (Kubernetes Operator)

A Kubernetes-native AI inference control plane built using Go and controller-runtime.

This project extends the Kubernetes API with a custom resource (`AIDeployment`) that declaratively manages AI model inference workloads, including Deployment orchestration, Service exposure, autoscaling, drift detection, and observability.

---

## Overview

The AI Inference Orchestrator introduces a new Kubernetes resource:

**AIDeployment**

This custom resource allows users to declaratively define:

- Model name
- Replica count
- Container port
- Service type (ClusterIP / NodePort / LoadBalancer)
- CPU / Memory resource requirements
- Optional container image override
- Horizontal autoscaling configuration

The controller continuously reconciles the desired state into:

- A Kubernetes Deployment
- A Kubernetes Service
- A Kubernetes HorizontalPodAutoscaler (optional)

The operator follows Kubernetes controller best practices:

- Idempotent reconciliation
- Drift detection
- Immutable field safety
- HPA-safe replica management
- Conflict-safe status updates
- Prometheus-native metrics exposure

---


## Architecture

User → AIDeployment (CRD)
↓
Controller → Reconcile Loop
↓
Deployment + Service + HPA
↓
Status Propagation
↓
Prometheus Metrics

The controller:

- Watches `AIDeployment`
- Owns `Deployment`
- Owns `Service`
- Owns `HorizontalPodAutoscaler`
- Performs safe drift detection
- Preserves HPA-managed replicas
- Updates only mutable fields
- Propagates Deployment readiness into CRD status
- Exposes reconciliation and business metrics
---

## API Specification

Example:

```yaml
apiVersion: infra.example.com/v1
kind: AIDeployment
metadata:
  name: test-model
spec:
  model: tinyllama
  replicas: 1
  port: 8080
  serviceType: ClusterIP
  resources:
    requests:
      cpu: "1000m"
      memory: "1Gi"
    limits:
      cpu: "2000m"
      memory: "2Gi"
  autoscaling:
    enabled: true
    minReplicas: 1
    maxReplicas: 3
    targetCPUUtilization: 60
```

### Autoscaling (Week 2)

The operator supports Kubernetes Horizontal Pod Autoscaler (autoscaling/v2).

When autoscaling is enabled:

The controller does not overwrite Deployment replicas

HPA manages replica count

CPU utilization drives scaling

Stabilization windows are applied:

Scale up: 30 seconds

Scale down: 120 seconds

The HPA is automatically created and owned by the AIDeployment resource.

### Observability (Week 2)

The controller exposes Prometheus-compatible metrics on the controller-runtime metrics endpoint.

Reconciliation Metrics

aideployment_reconcile_total

aideployment_reconcile_errors_total

aideployment_reconcile_duration_seconds

Business Metrics

aideployment_active_total

aideployment_replicas_current

aideployment_replicas_available

aideployment_autoscaling_enabled

These metrics allow monitoring of:

Controller health

Reconcile latency

Error rates

Deployment health

Autoscaling state


## Local Development

Install CRD into cluster:

```
make install
```
Run controller locally:
```
make run
```
Apply test resource:
```
kubectl apply -f test-aideployment.yaml
```
Inspect resources:
```
kubectl get deployments
kubectl get svc
kubectl get hpa
kubectl describe aideployment test-model
curl http://localhost:8080/metrics
```

## Features 

### v0.1.0

Custom CRD with OpenAPI schema validation

Status subresource enabled

Deployment reconciliation

Service reconciliation

Drift detection (replicas, image, resources)

Immutable-safe update logic

Condition propagation from Deployment

Kubernetes Event emission

Controller-runtime based architecture

CI integration (lint + controller tests)

### v0.2.0

Horizontal Pod Autoscaler integration

HPA-safe reconciliation logic

Autoscaling behavior tuning

Conflict-safe status updates

Prometheus metrics integration

Business-level metrics exposure

Verified autoscaling under load

## Roadmap

### Week 3

Prometheus + ServiceMonitor integration

Grafana dashboard

CLI for model deployment

MCP (Model Context Protocol) server

Plain-English model deployment interface

### Week 4

Advanced scaling policies

Custom metrics autoscaling

AI workload scheduling enhancements

Production hardening

Multi-tenant support

### Project Goals

The long-term goal of this project is to build a Kubernetes-native AI inference control plane that:

Treats AI models as first-class Kubernetes resources

Provides autoscaling tailored for AI workloads

Exposes rich observability signals

Enables declarative, policy-driven AI deployment

Integrates with CLI and natural language interfaces