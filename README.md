# AI Inference Orchestrator (Kubernetes Operator)

A Kubernetes-native AI inference control plane built using **Go** and **controller-runtime**.

This project extends the Kubernetes API with a custom resource **`AIDeployment`** that declaratively manages AI model inference workloads, including deployment orchestration, service exposure, autoscaling, drift detection, and observability.

The platform also exposes infrastructure controls through a **CLI**, an **MCP tool server**, and a **natural language interface**, enabling AI model lifecycle management through both traditional and AI-driven workflows.

---

# Overview

The AI Inference Orchestrator introduces a new Kubernetes resource:

**`AIDeployment`**

This custom resource allows users to declaratively define:

* AI model name
* Replica count
* Container port
* Service type (`ClusterIP`, `NodePort`, `LoadBalancer`)
* CPU / memory resource requirements
* Optional container image override
* Horizontal autoscaling configuration

The controller continuously reconciles the desired state into:

* A Kubernetes **Deployment**
* A Kubernetes **Service**
* A Kubernetes **HorizontalPodAutoscaler** (optional)

The operator follows Kubernetes controller best practices:

* Idempotent reconciliation
* Drift detection
* Immutable field safety
* HPA-safe replica management
* Conflict-safe status updates
* Prometheus-native metrics exposure

---

# Architecture

The system introduces multiple layers that interact to manage AI inference workloads.

```
User / CLI / Agent
        │
        ▼
MCP Tool Server
        │
        ▼
AIDeployment (Custom Resource)
        │
        ▼
Controller Reconcile Loop
        │
        ▼
Deployment + Service + HPA
        │
        ▼
Status Propagation
        │
        ▼
Prometheus Metrics
```

The controller:

* Watches `AIDeployment`
* Owns `Deployment`
* Owns `Service`
* Owns `HorizontalPodAutoscaler`
* Performs safe drift detection
* Preserves HPA-managed replicas
* Updates only mutable fields
* Propagates Deployment readiness into CRD status
* Exposes reconciliation and business metrics

---

# API Specification

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

---

# Autoscaling (Week 2)

The operator supports Kubernetes **Horizontal Pod Autoscaler** (`autoscaling/v2`).

When autoscaling is enabled:

* The controller does **not overwrite Deployment replicas**
* HPA manages replica count
* CPU utilization drives scaling

Stabilization windows are applied:

Scale up: **30 seconds**
Scale down: **120 seconds**

The HPA is automatically created and owned by the `AIDeployment` resource.

---

# Observability (Week 2)

The controller exposes **Prometheus-compatible metrics** via the controller-runtime metrics endpoint.

### Reconciliation Metrics

```
aideployment_reconcile_total
aideployment_reconcile_errors_total
aideployment_reconcile_duration_seconds
```

### Business Metrics

```
aideployment_active_total
aideployment_replicas_current
aideployment_replicas_available
aideployment_autoscaling_enabled
```

These metrics allow monitoring of:

* Controller health
* Reconciliation latency
* Error rates
* Deployment readiness
* Autoscaling state

---

# CLI Interface

A CLI tool provides a simplified interface for managing AI deployments.

Example commands:

```
aictl deploy mistral --replicas 3
aictl scale mistral --replicas 5
aictl endpoint mistral
aictl logs mistral
aictl list
```

The CLI communicates directly with Kubernetes using the controller-runtime client.

---

# MCP Tool Server

The project includes an **MCP (Model Context Protocol) server** exposing infrastructure capabilities as tools that can be invoked by AI agents.

Available tools:

```
deploy_model
scale_model
delete_model
list_models
model_status
```

Example request:

```
curl -X POST localhost:8085/tools \
-H "Content-Type: application/json" \
-d '{
  "tool": "deploy_model",
  "args": {
    "model": "mistral",
    "replicas": 3
  }
}'
```

---

# Natural Language Interface

A simple agent endpoint allows infrastructure operations using natural language.

Example prompts:

```
deploy mistral with 3 replicas
scale mistral to 5 replicas
status mistral
delete mistral
```

The agent parses the request and invokes MCP tools accordingly.

---

# Local Development

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
```

View metrics:

```
curl http://localhost:8080/metrics
```

---

# Features

## v0.1.0

* Custom CRD with OpenAPI schema validation
* Status subresource enabled
* Deployment reconciliation
* Service reconciliation
* Drift detection (replicas, image, resources)
* Immutable-safe update logic
* Condition propagation from Deployment
* Kubernetes Event emission
* Controller-runtime based architecture
* CI integration (lint + controller tests)

## v0.2.0

* Horizontal Pod Autoscaler integration
* HPA-safe reconciliation logic
* Autoscaling behavior tuning
* Conflict-safe status updates
* Prometheus metrics integration
* Business-level metrics exposure
* Verified autoscaling under load

## v1.0
* Added CLI and MCP server support
* Fixed reconciliation logic

---

# Project Goals

The long-term goal of this project is to build a **Kubernetes-native AI inference control plane** that:

* Treats AI models as **first-class Kubernetes resources**
* Enables **declarative, policy-driven AI deployment**
* Supports **autoscaling tailored for AI workloads**
* Integrates with **CLI and AI-driven orchestration interfaces**
* Provides **observability and operational tooling for AI infrastructure**
