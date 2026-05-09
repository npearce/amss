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

Build and test all services individually.

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

Expected: all pass with no race conditions.

| Service | Tests |
|---|---|
| kb-store | 47 |
| ticket-store | 88 |
| crew-store | 90 |
| **Stores total** | **225** |

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

MCP servers are scaffolded with `kmcp` and follow the kagent MCP tool pattern.

**Scaffold (already done — only needed when creating from scratch):**

```bash
cd mcp-servers
kmcp init go kb-mcp --go-module-name github.com/npearce/amss/mcp-servers/kb-mcp --no-git
kmcp init go ticket-mcp --go-module-name github.com/npearce/amss/mcp-servers/ticket-mcp --no-git
```

**Build and test:**

```bash
cd mcp-servers/kb-mcp
go build ./...
go test ./... -v -race
go vet ./...
cd ../..
```

```bash
cd mcp-servers/ticket-mcp
go build ./...
go test ./... -v -race
go vet ./...
cd ../..
```

| Service | Tests |
|---|---|
| kb-mcp | 45 |
| ticket-mcp | 61 |
| **MCP total** | **106** |

**Run locally (stdio mode for testing with MCP inspector):**

```bash
cd mcp-servers/kb-mcp
KB_STORE_URL=http://localhost:8081 go run ./cmd/server/
```

**Run locally (HTTP mode):**

```bash
cd mcp-servers/kb-mcp
KB_STORE_URL=http://localhost:8081 go run ./cmd/server/ -http=:9001
```

Note: The kb-store must be running on :8081 for the MCP server to function.

### 1.4 Build and Test BFF

```bash
cd bff
go build ./...
go test ./... -v -race
go vet ./...
cd ..
```

103 tests covering proxy routes (root-path and `/api/v1/` variants), stub chat and curator responses, CORS, reset, and agent fallback behavior.

### 1.5 Agents

Agents are kagent `Agent` CRDs — YAML manifests, not Go services. They live in `agents/` and have no local build step. They are applied to the cluster in Phase 3 once kagent is installed.

```
agents/
├── mission-support-agent/
│   ├── agent.yaml        # kagent Agent CRD
│   └── prompts/system.md
└── kb-curator-agent/
    ├── agent.yaml
    └── prompts/system.md
```

### 1.6 Build Frontend

```bash
cd frontend
npm install
npm run build   # produces dist/ — verified clean in CI
cd ..
```

In dev, `npm run dev` starts Vite at `http://localhost:5173`. All `/api/v1/*` requests are proxied to the BFF at `localhost:8080`.

### 1.7 Build and Test Activity Generator

```bash
cd activity-generator
go build ./...
go test ./... -v -race
go vet ./...
cd ..
```

17 tests covering scenario loading, template resolution, step execution, and multi-step chaining.

Run against a live BFF:
```bash
cd activity-generator
BFF_URL=http://localhost:30080 go run .
```

Runs 4 scenarios — creates a P3 ticket, creates and closes a P4 ticket, publishes a KB article, publishes and archives a superseded KB article — then exits.

---

## Phase 2 — Kubernetes (OrbStack, No Solo Products Yet)

Deploy the full AMSS application to a local k8s cluster. Agents run in stub mode (keyword-matched responses, no LLM required).

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
# Uncomment the kind load lines in k8s/deploy.sh before running
```

### 2.2 Deploy (One Command)

Run from the **repo root**:

```bash
./k8s/deploy.sh
```

The script:
1. Builds 7 Docker images: `amss/kb-store`, `amss/ticket-store`, `amss/crew-store`, `amss/kb-mcp`, `amss/ticket-mcp`, `amss/bff`, `amss/frontend`
2. The frontend image is built with `VITE_API_URL=http://localhost:30080` baked in (Vite embeds it at bundle time)
3. Applies manifests in order: namespace → stores → MCP servers → BFF → frontend
4. Waits for all 7 deployments to reach Ready
5. Prints access URLs

On success:
```
==> All deployments ready.

  Frontend:  http://localhost:30081
  BFF API:   http://localhost:30080
```

### 2.3 Verify

**BFF health:**
```bash
curl -s http://localhost:30080/health
# {"data":{"status":"ok","service":"bff"},"error":null}
```

**Crew store (20 members):**
```bash
curl -s http://localhost:30080/api/v1/crew | jq '.data.total'
# 20
```

**KB store (30 articles):**
```bash
curl -s http://localhost:30080/api/v1/kb | jq '.data.total'
# 30
```

**Ticket store (15 tickets):**
```bash
curl -s http://localhost:30080/api/v1/tickets | jq '.data.total'
# 15
```

**Chat (stub mode — no agent required):**
```bash
curl -s -X POST http://localhost:30080/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"test-001","mission":"artemis-ii","message":"WCS pressure is dropping"}' \
  | jq '.data'
```

Expected response references KB-001 (the WCS toilet pressure fault article).

**Frontend:**
```bash
open http://localhost:30081
```

Two views: Astronaut Chat (`/chat`) and Ground Control (`/ground-control`). Use the user switcher to change crew member identity.

**KB curation (stub mode):**
```bash
curl -s -X POST http://localhost:30080/api/v1/curator \
  -H 'Content-Type: application/json' \
  -d '{}' \
  | jq '.data.duplicates_flagged'
```

Returns 3 near-duplicate WCS articles flagged against KB-001.

**Reset all stores to seed data:**
```bash
curl -s -X POST http://localhost:30080/api/v1/reset | jq '.data'
```

### 2.4 Run Activity Generator Against k8s

```bash
cd activity-generator
BFF_URL=http://localhost:30080 go run .
```

Fires 4 scenarios against the live BFF. Verify tickets and articles were created:
```bash
curl -s http://localhost:30080/api/v1/tickets | jq '.data.total'
# 17 (15 seed + 2 created)
curl -s http://localhost:30080/api/v1/kb | jq '.data.total'
# 32 (30 seed + 2 created; one immediately archived)
```

### 2.5 Test Summary (Phase 2 Complete)

| Service | Tests |
|---|---|
| kb-store | 47 |
| ticket-store | 88 |
| crew-store | 90 |
| kb-mcp | 45 |
| ticket-mcp | 61 |
| bff | 103 |
| activity-generator | 17 |
| **Total** | **451** |

All pass with `-race` flag. Run the full suite from the repo root:
```bash
for svc in stores/kb-store stores/ticket-store stores/crew-store \
           mcp-servers/kb-mcp mcp-servers/ticket-mcp \
           bff activity-generator; do
  echo "==> $svc"
  (cd "$svc" && go test ./... -race)
done
```

### 2.6 Clean Redeploy

To wipe and redeploy from scratch:
```bash
./k8s/teardown.sh && ./k8s/deploy.sh
```

`teardown.sh` deletes the `amss` namespace (all resources). `deploy.sh` rebuilds images and redeploys.

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

Apply the pre-authored Agent CRDs:
```bash
kubectl apply -f agents/mission-support-agent/agent.yaml -n amss
kubectl apply -f agents/kb-curator-agent/agent.yaml -n amss
```

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

### 3.6 Flip BFF Out of Stub Mode

Once agents are deployed and reachable:
```bash
kubectl set env deployment/bff STUB_MODE=false -n amss
kubectl rollout status deployment/bff -n amss
```

### 3.7 Verify Full Stack

> **TODO**: End-to-end verification steps.

```bash
# Access kagent Enterprise UI
kubectl port-forward service/kagent-enterprise-ui -n kagent 4000:80 &
open http://localhost:4000

# Test chat via BFF (now routed to live agent)
curl -s -X POST http://localhost:30080/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"demo-1","mission":"artemis-ii","message":"What is the WCS flush procedure?"}' \
  | jq '.data.response'
```

---

## Phase 4 — Demo Walkthrough

> **TODO**: Scripted demo narrative showing before/after with Solo products.

---

## Teardown

### Remove AMSS Application

```bash
./k8s/teardown.sh
```

This deletes the `amss` namespace and every resource inside it (deployments, services, pods, secrets).

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

### BFF returns 404 for /api/v1/* routes

The BFF handles both root-path routes (for local dev via Vite proxy) and `/api/v1/` routes (for k8s). The `/api/v1/kb` path is rewritten to `/articles` at the kb-store — confirm with:
```bash
curl -s http://localhost:30080/api/v1/kb/KB-001 | jq '.data.title'
```

### Frontend can't reach BFF in k8s

`VITE_API_URL` is embedded at Docker build time. Rebuilding the frontend image after changing the URL requires a full redeploy:
```bash
./k8s/teardown.sh && ./k8s/deploy.sh
```

In dev, `VITE_API_URL` is unset and the Vite proxy handles `/api/v1/*` → `localhost:8080`.

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
| 2026-05-09 | Phase 1 | Initial runbook. Three stores built and tested (225 tests). |
| 2026-05-09 | Phase 2 | MCP servers (106 tests), BFF with stub mode (103 tests), React frontend, activity generator (17 tests). Full k8s deploy via `./k8s/deploy.sh`. 451 total tests. Frontend at localhost:30081, BFF at localhost:30080. |
