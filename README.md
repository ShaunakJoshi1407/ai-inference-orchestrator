# AI Inference Orchestrator (Kubernetes Operator)

A Kubernetes-native AI model deployment control plane built using Go and controller-runtime.

This project extends the Kubernetes API with a custom resource (`AIDeployment`) that declaratively manages AI model inference workloads, including Deployment and Service orchestration, drift detection, and status propagation.

---

## Overview

The AI Inference Orchestrator introduces a new Kubernetes resource:

AIDeployment

This custom resource allows users to declaratively specify:

- Model name
- Replica count
- Container port
- Service type (ClusterIP / NodePort / LoadBalancer)
- Optional image override
- CPU / Memory resource requirements

The controller continuously reconciles the desired state into:

- A Kubernetes Deployment
- A Kubernetes Service

The operator follows Kubernetes controller best practices, including idempotent reconciliation, immutable field safety, and status condition propagation.

---

## Architecture

User → AIDeployment (CRD)  
Controller → Reconcile Loop  
Reconcile → Deployment + Service  
Status → Conditions updated from Deployment  
Events → Emitted via EventRecorder  

The controller:

- Watches AIDeployment
- Owns Deployment
- Owns Service
- Performs safe drift detection
- Updates only mutable fields
- Propagates Deployment conditions into CRD status

---

## API Specification

Example:

```yaml
apiVersion: infra.example.com/v1
kind: AIDeployment
metadata:
  name: test-model
spec:
  model: llama3
  replicas: 4
  port: 8080
  serviceType: ClusterIP
  resources:
    requests:
      cpu: "100m"
      memory: "128Mi"
    limits:
      cpu: "500m"
      memory: "256Mi"
```

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
kubectl describe aideployment test-model
```

## Features (v0.1.0)

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

## Roadmap

### Week 2

Horizontal Pod Autoscaler integration

Metrics endpoint

Observability enhancements

### Week 3

CLI for model deployment

MCP (Model Context Protocol) server

Plain-English model deployment interface

### Week 4

Advanced scaling policies

AI workload scheduling enhancements

Production hardening