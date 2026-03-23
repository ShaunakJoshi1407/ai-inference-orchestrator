# AI Inference Orchestrator

A Kubernetes-native AI inference control plane built with **Go** and **controller-runtime**.

This project extends the Kubernetes API with a custom resource — **`AIDeployment`** — that declaratively manages AI model inference workloads, including deployment orchestration, service exposure, autoscaling, drift detection, and observability.

Infrastructure controls are also exposed through a **CLI**, an **HTTP tool server**, and a **natural language interface**, enabling model lifecycle management through both traditional and agent-driven workflows.

---

## Overview

The `AIDeployment` custom resource lets you declaratively define:

- AI model name and optional container image override
- Replica count
- Service port and type (`ClusterIP`, `NodePort`, `LoadBalancer`)
- CPU / memory resource requirements
- Horizontal autoscaling configuration

The controller continuously reconciles the desired state into:

- A Kubernetes **Deployment**
- A Kubernetes **Service**
- A Kubernetes **HorizontalPodAutoscaler** (when autoscaling is enabled)

It follows Kubernetes controller best practices:

- Idempotent reconciliation — safe to run repeatedly
- Drift detection — updates owned resources when spec changes
- HPA-safe replica management — never overwrites replicas managed by autoscaler
- Conflict-safe status updates via retry-on-conflict
- Owner references — child resources are garbage-collected when the CR is deleted

---

## Architecture

```
┌─────────────────────────────────────────┐
│         User Interfaces                 │
│  aictl CLI  │  HTTP Tool Server  │  NL  │
└──────┬──────┴────────┬──────────┴───┬───┘
       │               │              │
       └───────────────▼──────────────┘
                AIDeployment CR
                       │
                       ▼
          Controller Reconcile Loop
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
      Deployment    Service        HPA
                       │
          ┌────────────┘
          ▼
  Status Propagation → Prometheus Metrics
```

The controller:

- Watches `AIDeployment` resources
- Owns and reconciles `Deployment`, `Service`, and `HPA`
- Performs drift detection — updates resources when spec changes
- Preserves HPA-managed replica counts when autoscaling is enabled
- Propagates Deployment readiness into CRD status conditions
- Exposes reconciliation and business metrics

---

## API Specification

```yaml
apiVersion: infra.example.com/v1
kind: AIDeployment
metadata:
  name: tinyllama
spec:
  model: tinyllama          # model name — passed to the container as MODEL env var
  image: ollama/ollama:latest  # optional image override
  replicas: 1
  port: 8080                # service port (target port is always 11434)
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

### Field behavior

| Field | Default | Notes |
|-------|---------|-------|
| `model` | required | Passed as `MODEL` env var to the container |
| `image` | `ollama/ollama:latest` | Override the container image |
| `replicas` | `1` | Ignored when `autoscaling.enabled: true` |
| `port` | `8080` | Exposed service port; container always listens on `11434` |
| `serviceType` | `ClusterIP` | Standard Kubernetes service types supported |
| `resources` | 200m/256Mi req, 500m/512Mi lim | Container resource requests and limits |

---

## Autoscaling

The operator supports Kubernetes **HorizontalPodAutoscaler** (`autoscaling/v2`).

When autoscaling is enabled:

- The controller does **not overwrite Deployment replicas** — the HPA manages replica count
- CPU utilization drives scaling decisions

Stabilization windows:

| Direction | Window |
|-----------|--------|
| Scale up | 30 seconds |
| Scale down | 120 seconds |

The HPA is automatically created and garbage-collected via owner references on the `AIDeployment` resource.

---

## Observability

The controller exposes **Prometheus-compatible metrics** via the controller-runtime metrics endpoint.

### Reconciliation metrics

```
aideployment_reconcile_total{name, namespace}
aideployment_reconcile_errors_total{name, namespace}
aideployment_reconcile_duration_seconds{name, namespace}
```

### Business metrics

```
aideployment_active_total
aideployment_replicas_current{name, namespace}
aideployment_replicas_available{name, namespace}
aideployment_autoscaling_enabled{name, namespace}
```

---

## CLI

`aictl` provides a simplified interface for managing AI deployments directly against the Kubernetes API.

```bash
aictl deploy mistral --replicas 3
aictl scale mistral --replicas 5
aictl endpoint mistral
aictl logs mistral
aictl list
```

---

## Tool Server

The project includes an HTTP tool server that exposes infrastructure operations as callable tools — suitable for consumption by AI agents or scripts.

**Endpoints:**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/tools` | POST | Invoke a tool by name |
| `/tools/list` | GET | List available tools |
| `/agent` | POST | Natural language interface |

**Available tools:**

| Tool | Args | Description |
|------|------|-------------|
| `deploy_model` | `model`, `replicas` | Create a new AIDeployment |
| `scale_model` | `model`, `replicas` | Update replica count |
| `delete_model` | `model` | Delete an AIDeployment |
| `list_models` | — | List all AIDeployments |
| `model_status` | `model` | Get replicas and availability |

**Example:**

```bash
curl -X POST localhost:8085/tools \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "deploy_model",
    "args": { "model": "mistral", "replicas": 3 }
  }'
```

> **Note:** This is a JSON HTTP API, not the standard MCP JSON-RPC 2.0 protocol. Migrating to standard MCP transport (stdio/SSE) would allow native integration with MCP-compatible clients such as Claude Desktop.

---

## Natural Language Interface

The `/agent` endpoint accepts free-text prompts and maps them to tool calls.

```bash
curl -X POST localhost:8085/agent \
  -d "deploy mistral with 3 replicas"

curl -X POST localhost:8085/agent \
  -d "scale mistral to 5 replicas"

curl -X POST localhost:8085/agent \
  -d "status mistral"
```

> **Note:** Intent parsing is currently regex-based. A natural extension would be to back it with an LLM call (e.g. Claude API via `ANTHROPIC_API_KEY`) to handle broader phrasing and ambiguous inputs, with the regex parser as a fallback.

---

## Testing

The controller is tested using **envtest** — a real Kubernetes API server and etcd spun up in-process, with no cluster required. Tests are written in Ginkgo (BDD style).

```bash
# Install envtest binaries (first time only)
make setup-envtest

# Run the test suite
make test
```

### Test dry run

```
Running Suite: Controller Suite
===============================
Will run 11 of 11 specs

AIDeployment Controller Deployment creation
  uses the default image when spec.image is not set       • [2.031s]
  uses spec.image when provided                           • [0.010s]
  sets the MODEL env var from spec.model                  • [0.007s]

AIDeployment Controller Service creation
  uses the default port (8080) when spec.port is not set  • [0.006s]
  uses spec.port when provided                            • [0.006s]
  uses the default serviceType (ClusterIP) when not set   • [0.006s]
  uses spec.serviceType when provided                     • [0.007s]
  reconciles service port drift when spec.port changes    • [0.010s]

AIDeployment Controller Basic reconciliation
  creates owned Deployment and Service for a minimal spec • [0.005s]
  does not return an error for a missing resource         • [0.000s]
  is idempotent — reconciling twice produces no error     • [0.008s]

Ran 11 of 11 Specs in 5.835 seconds
SUCCESS! -- 11 Passed | 0 Failed | 0 Pending | 0 Skipped
```

### What is tested

| Area | What is verified |
|------|-----------------|
| Default image | Container image is `ollama/ollama:latest` when `spec.image` is unset |
| Image override | `spec.image` is applied to the container |
| Model env var | `MODEL` env var is set from `spec.model` |
| Default port | Service port is `8080` when `spec.port` is unset |
| Port override | `spec.port` is applied to the Service |
| Default serviceType | Service type is `ClusterIP` when `spec.serviceType` is unset |
| ServiceType override | `spec.serviceType` is applied to the Service |
| Service drift | Changing `spec.port` on an existing CR updates the Service |
| Minimal spec | Reconcile creates both Deployment and Service for a bare-minimum CR |
| Missing resource | Reconciling a non-existent resource returns no error |
| Idempotency | Reconciling the same CR twice produces no error or duplicate resources |

---

## Local Development

**Prerequisites:** Go 1.25+, kubectl, a running Kubernetes cluster (kind works well)

```bash
# Install CRD into cluster
make install

# Run controller locally
make run

# Apply a test resource
kubectl apply -f test-aideployment.yaml

# Inspect managed resources
kubectl get deployments
kubectl get svc
kubectl get hpa
kubectl describe aideployment tinyllama

# View Prometheus metrics
curl http://localhost:8080/metrics

# Run the tool server
go run ./mcp
```

---

## Capabilities

- Custom CRD (`AIDeployment`) with OpenAPI schema validation
- Deployment, Service, and HPA reconciliation with drift detection
- Idempotent, conflict-safe controller following Kubernetes best practices
- HPA integration with stabilization windows tuned for AI workloads
- Status conditions propagated from underlying Deployment readiness
- Prometheus metrics for reconciliation health and business state
- `aictl` CLI for direct cluster management
- HTTP tool server exposing 5 model lifecycle operations
- Natural language agent endpoint backed by keyword-based intent parsing
- Owner references for automatic garbage collection of child resources
