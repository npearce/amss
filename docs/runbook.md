# AMSS Runbook

Reproduction guide for the Artemis Mission Support System demo. Follow these steps to go from a clean machine to a running demo.

---

## Prerequisites

### Required Tools

| Tool | Version | Install | Purpose |
|---|---|---|---|
| Go | 1.22+ | [go.dev/dl](https://go.dev/dl/) | Backend services |
| Node.js | 20+ | [nodejs.org](https://nodejs.org/) | Frontend (React + Vite) |
| Docker | 24+ | Comes with OrbStack | Container builds |
| OrbStack | Latest | [orbstack.dev](https://orbstack.dev/) | Local k8s (preferred over kind) |
| kubectl | 1.29+ | Comes with OrbStack | Kubernetes CLI |
| Helm | 3.14+ | `brew install helm` | Package management for k8s |
| kmcp | Latest | See below | MCP server scaffolding and deployment |

### kmcp Install

```bash
curl -fsSL https://raw.githubusercontent.com/kagent-dev/kmcp/refs/heads/main/scripts/get-kmcp.sh | bash
```

Verify:
```bash
kmcp version
```

### License Keys

Contact your Solo account representative to obtain:

```bash
export SOLO_ISTIO_LICENSE_KEY=<key>
export GLOO_GATEWAY_LICENSE_KEY=<key>
export AGENTGATEWAY_LICENSE_KEY=<key>
```

Store these securely. Do not commit them to the repo.

### LLM API Key

At least one LLM provider key is required for agents:

```bash
export OPENAI_API_KEY=<key>
# or
export ANTHROPIC_API_KEY=<key>
```

### Clone the Repo

```bash
git clone https://github.com/npearce/amss.git
cd amss
git checkout v0.1.0-alpha1  # or the target branch
```

---

## Phase 1 — Local Dev (No k8s, No Solo Products)

Build and test all services individually, then run them together with docker-compose.

### 1.1 Build and Test Stores

Each store is an independent Go module. Build and test them one at a time.

**KB Store:**
```bash
cd stores/kb-store
go build ./...
go test ./... -v -race
go vet ./...
cd ../..
```

**Ticket Store:**
```bash
cd stores/ticket-store
go build ./...
go test ./... -v -race
go vet ./...
cd ../..
```

**Crew Store:**
```bash
cd stores/crew-store
go build ./...
go test ./... -v -race
go vet ./...
cd ../..
```

Expected: All three stores pass with 200+ tests total, zero race conditions.

### 1.2 Run a Store Locally

Any store can be run standalone for manual testing:

```bash
cd stores/kb-store
SEED_PATH=seed-data/kb.json PORT=8081 go run .
```

In another terminal:
```bash
# Health check
curl http://localhost:8081/health

# List articles
curl http://localhost:8081/articles | jq .

# Search
curl "http://localhost:8081/articles?search=pressure+fault" | jq .

# Get by ID
curl http://localhost:8081/articles/KB-001 | jq .
```

### 1.3 Build and Test MCP Servers

> **TODO**: MCP servers are being rebuilt using `kmcp init go`. This section will be updated with the kmcp-scaffolded workflow.

### 1.4 Build and Test BFF

```bash
cd bff
go build ./...
go test ./... -v -race
go vet ./...
cd ..
```

### 1.5 Build and Test Agents

> **TODO**: Agents will be built as kagent `Agent` CRDs. Local dev stubs TBD.

### 1.6 Build Frontend

> **TODO**: React + Vite frontend.

### 1.7 Docker Compose — Full Local Stack

> **TODO**: docker-compose.yml that runs all services together without Solo products.

```bash
docker compose up --build
```

Verify:
```bash
# BFF health (should report status of all downstream services)
curl http://localhost:8080/api/v1/health | jq .
```

---

## Phase 2 — Kubernetes (OrbStack, No Solo Products Yet)

Deploy the AMSS application to a local k8s cluster before adding Solo Enterprise products.

### 2.1 Create Cluster

OrbStack provides a built-in k8s cluster:

```bash
# OrbStack k8s is available by default
kubectl config use-context orbstack
kubectl get nodes
```

If using kind instead:
```bash
kind create cluster --name amss
kubectl config use-context kind-amss
```

### 2.2 Build and Load Images

> **TODO**: Build all container images and load them into the local cluster.

```bash
# Example for one service:
cd stores/kb-store
docker build -t amss/kb-store:latest .
# OrbStack sees local images automatically
# For kind: kind load docker-image amss/kb-store:latest --name amss
```

### 2.3 Deploy Application

> **TODO**: Helm chart or raw manifests for the AMSS application.

### 2.4 Verify

> **TODO**: Port-forward and test endpoints.

---

## Phase 3 — Solo Enterprise Products

Install Solo Enterprise for kagent (includes ambient mesh and agentgateway as waypoint) on top of the running k8s cluster.

### 3.1 Install Solo Enterprise for kagent

Follow: https://docs.solo.io/kagent-enterprise/docs/latest/quickstart/

> **TODO**: Document the exact commands as we run them. The quickstart uses kind but we're using OrbStack — note any differences.

Key steps (to be filled in with exact commands):
1. Install Istio ambient mesh (Solo distribution)
2. Install Solo Enterprise for agentgateway
3. Install Solo Enterprise for kagent
4. Set up OIDC (Keycloak for demo)
5. Verify all pods running

### 3.2 Install Solo Enterprise for agentgateway

Follow: https://docs.solo.io/agentgateway/2.3.x/install/helm

> **TODO**: Document exact Helm commands.

### 3.3 Deploy MCP Servers via kmcp

> **TODO**: Document `kmcp deploy` commands for kb-mcp and ticket-mcp.

The MCP servers become `MCPServer` CRDs managed by the kmcp controller:

```yaml
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata:
  name: kb-mcp
  namespace: amss
spec:
  deployment:
    image: amss/kb-mcp:latest
    port: 3000
  transportType: stdio
```

### 3.4 Deploy Agents via kagent

> **TODO**: Document Agent CRD creation.

Agents reference MCP servers by name:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: mission-support-agent
  namespace: amss
spec:
  type: Declarative
  declarative:
    modelConfig: default-model-config
    systemMessage: |
      <contents of prompts/system.md>
    tools:
      - type: McpServer
        mcpServer:
          name: kb-mcp
          kind: MCPServer
          toolNames:
            - search_kb
            - read_kb_article
      - type: McpServer
        mcpServer:
          name: ticket-mcp
          kind: MCPServer
          toolNames:
            - search_tickets
            - create_ticket
```

### 3.5 Configure agentgateway

> **TODO**: Document listener and route configuration for:
> - Ingress: external traffic → BFF
> - Egress: agent → LLM providers (with guardrails, failover)

### 3.6 Verify Full Stack

> **TODO**: End-to-end verification steps.

```bash
# Access kagent Enterprise UI
kubectl port-forward service/kagent-enterprise-ui -n kagent 4000:80 &
open http://localhost:4000

# Test chat via BFF
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "demo-1", "message": "What is the WCS flush procedure?"}'
```

---

## Phase 4 — Demo Walkthrough

> **TODO**: Scripted demo narrative showing before/after with Solo products.

---

## Teardown

### Remove AMSS Application

> **TODO**: Helm uninstall or kubectl delete commands.

### Remove Solo Enterprise Products

> **TODO**: Follow uninstall docs in reverse order of install.

### Remove Cluster

```bash
# OrbStack: cluster persists across restarts, reset via OrbStack app
# kind:
kind delete cluster --name amss
```

---

## Troubleshooting

### Store tests fail with "seed file not found"

The `SEED_PATH` env var must point to the seed-data JSON file relative to where you're running the command. When running from the service directory:

```bash
SEED_PATH=seed-data/kb.json go run .
```

### Docker build fails with "COPY with more than one source"

Ensure Dockerfiles don't reference `go.sum` if the module has no external dependencies. Fix: remove `go.sum` from the COPY line.

### OrbStack k8s not responding

```bash
# Check OrbStack is running
orb status

# Restart k8s
orb restart k8s
```

### kmcp deploy fails

Ensure the kmcp controller CRDs are installed:

```bash
kmcp install
kubectl get crd mcpservers.kagent.dev
```

---

## Version History

| Date | Phase | What Changed |
|---|---|---|
| 2026-05-09 | Phase 1 | Initial runbook. Three stores built and tested (225+ tests). |