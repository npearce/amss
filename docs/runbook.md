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
export AGENTGATEWAY_LICENSE_KEY=<key>   # Phase 3
export SOLO_ISTIO_LICENSE_KEY=<key>     # Phase 4
export GLOO_GATEWAY_LICENSE_KEY=<key>   # Phase 4
```

Store these securely. Do not commit them to the repo.

### LLM API Key

Required for agents (Phase 4):

```bash
export ANTHROPIC_API_KEY=<key>
```

### Clone the Repo

```bash
git clone https://github.com/npearce/amss.git
cd amss
git checkout v0.1.0-alpha2  # or the target branch
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
curl http://localhost:8081/health
curl http://localhost:8081/articles | jq .
curl "http://localhost:8081/articles?search=pressure+fault" | jq .
curl http://localhost:8081/articles/KB-001 | jq .
```

### 1.3 Build and Test MCP Servers

MCP servers are scaffolded with `kmcp init go --no-git` and follow the kagent MCP tool pattern.

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

Note: the kb-store must be running on `:8081` for the MCP server to function.

### 1.4 Build and Test BFF

```bash
cd bff
go build ./...
go test ./... -v -race
go vet ./...
cd ..
```

103 tests covering proxy routes (root-path and `/api/v1/` variants), stub chat and curator responses, CORS, reset, and agent fallback behavior. Stub mode includes 12 keyword patterns that return realistic canned responses citing real KB article IDs.

### 1.5 Agents

Agents are kagent `Agent` CRDs — YAML manifests, not Go services. They live in `agents/` and have no local build step. They are applied to the cluster in Phase 4 once kagent is installed.

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
npm run build   # produces dist/ — verified clean
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

46 tests covering scenario loading (with `start_offset_seconds`, timing hints, `stash_as`), template resolution, `humanDelay` with context cancellation, `formatCrewLabel`, session ID generation, `mergeMaps`, step execution with body template substitution, and multi-step scenario chaining including the `stash_as` carry mechanism.

The generator runs 6 narrative scenarios concurrently in humanized 22-minute cycles. Each scenario simulates a crew member or ground control session with realistic pre- and post-step delays. After all scenarios complete (or the cycle timeout elapses), stores are reset to seed data and the cycle repeats.

The 6 scenarios:
1. **Morning systems check — Koch**: 2 chat steps querying ECLSS status
2. **Comms glitch investigation — Glover**: 2 chat steps + creates P2 comms ticket + gc-comm adds assessment comment
3. **EVA suit prep concern — Hansen**: 2 chat steps + creates P2 EVA ticket + gc-eva issues no-go ruling
4. **KB article update — gc-eclss**: Publishes CO2 scrubber replacement procedure + CAPCOM verification chat
5. **Checking on old issues — Wiseman**: 3 chat steps reviewing open tickets as mission commander
6. **Ticket resolution — gc-systems**: Creates WCS pressure ticket (stash_as: ticket_id) → in-progress → comment with RCA → closed

Run against a live BFF:
```bash
cd activity-generator
BFF_URL=http://localhost:8080 CYCLE_MINUTES=22 go run .
```

---

## Phase 2 — Kubernetes (OrbStack, No Solo Products Yet)

Deploy the full AMSS application to a local k8s cluster. Agents run in stub mode (keyword-matched responses, no LLM required).

### 2.1 Create Cluster

OrbStack provides a built-in k8s cluster:

```bash
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
2. The frontend image is built **without** `VITE_API_URL` — the SPA uses `/api/v1/...` as relative paths, which works when frontend and BFF are behind the same gateway host
3. Applies manifests in order: namespace → stores → MCP servers → BFF → frontend → agentgateway-routes.yaml (safe to apply before agentgateway is installed)
4. Waits for all 7 deployments to reach Ready
5. Prints access URLs

On success:
```
==> All deployments ready.

  agentgateway:  http://192.168.139.2        (frontend + BFF via gateway)
  Frontend:      http://localhost:30081       (NodePort direct — dev/debug only)
  BFF API:       http://localhost:30080       (NodePort direct — dev/debug only)
```

### 2.3 Verify

**BFF health (direct NodePort):**
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

**KB curation (stub mode):**
```bash
curl -s -X POST http://localhost:30080/api/v1/curator \
  -H 'Content-Type: application/json' \
  -d '{}' \
  | jq '.data.duplicates_flagged'
# 3
```

Returns 3 near-duplicate WCS articles flagged against KB-001.

**Reset all stores to seed data:**
```bash
curl -s -X POST http://localhost:30080/api/v1/reset | jq '.data'
```

> **NodePort note**: the frontend at `localhost:30081` makes relative `/api/v1/...` calls which resolve against `localhost:30081` — the frontend service, not the BFF. Direct NodePort access is BFF-only for manual API testing. Use agentgateway (Phase 3) for a working end-to-end UI flow, or `npm run dev` in local dev.

### 2.4 Run Activity Generator Against k8s

```bash
cd activity-generator
BFF_URL=http://localhost:30080 go run .
```

One cycle fires 6 scenarios concurrently against the live BFF. After the cycle the stores reset and it repeats. Press Ctrl+C to stop (graceful shutdown).

Verify one cycle landed:
```bash
curl -s http://localhost:30080/api/v1/tickets | jq '.data.total'
# 18 (15 seed + 3 created by scenarios 2, 3, 6)
curl -s http://localhost:30080/api/v1/kb | jq '.data.total'
# 31 (30 seed + 1 created by scenario 4)
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
| activity-generator | 46 |
| **Total** | **480** |

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

## Phase 3 — Solo Enterprise for agentgateway

Install Solo Enterprise agentgateway on top of the running k8s cluster. This puts the frontend and BFF behind a single gateway address with AI-native routing capabilities.

### 3.1 Install Gateway API CRDs

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.0/standard-install.yaml
```

### 3.2 Install Enterprise agentgateway

```bash
export AGENTGATEWAY_LICENSE_KEY=<your-key>

helm upgrade -i enterprise-agentgateway-crds \
  oci://us-docker.pkg.dev/solo-public/enterprise-agentgateway/charts/enterprise-agentgateway-crds \
  --create-namespace \
  --namespace agentgateway-system \
  --version v2.3.2

helm upgrade -i enterprise-agentgateway \
  oci://us-docker.pkg.dev/solo-public/enterprise-agentgateway/charts/enterprise-agentgateway \
  -n agentgateway-system \
  --version v2.3.2 \
  --set-string licensing.licenseKey=${AGENTGATEWAY_LICENSE_KEY}
```

### 3.3 Create the Gateway Proxy

```bash
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: amss-gateway
  namespace: agentgateway-system
spec:
  gatewayClassName: enterprise-agentgateway
  listeners:
  - protocol: HTTP
    port: 80
    name: http
    allowedRoutes:
      namespaces:
        from: All
EOF
```

Get the gateway IP (on OrbStack this returns a routable address like `192.168.139.2`):

```bash
kubectl get gateway amss-gateway -n agentgateway-system \
  -o jsonpath='{.status.addresses[0].value}'
```

### 3.4 Apply HTTPRoutes

The agentgateway routes are already applied by `deploy.sh`. They live in `k8s/agentgateway-routes.yaml` and create three resources:

| Resource | Kind | Purpose |
|---|---|---|
| `agentgateway-to-amss` | ReferenceGrant | Allows the Gateway in `agentgateway-system` to reference Services in `amss` |
| `bff-api-route` | HTTPRoute | Routes `/api/v1/*` → bff:8080 (higher priority — longer prefix) |
| `frontend-route` | HTTPRoute | Routes `/*` → frontend:80 (catch-all) |

If applying manually:
```bash
kubectl apply -f k8s/agentgateway-routes.yaml
```

Check attachment status:
```bash
kubectl get httproute -n amss -o wide
```

Both routes should show `Accepted` and `ResolvedRefs`.

### 3.5 Install Solo Enterprise UI

```bash
helm upgrade -i management \
  oci://us-docker.pkg.dev/solo-public/solo-enterprise-helm/charts/management \
  --namespace agentgateway-system \
  --version 0.3.19 \
  --set cluster="mgmt-cluster" \
  --set products.agentgateway.enabled=true \
  --set-string licensing.licenseKey=${AGENTGATEWAY_LICENSE_KEY}
```

Access the UI:
```bash
kubectl port-forward service/solo-enterprise-ui -n agentgateway-system 4000:80 &
open http://localhost:4000
```

### 3.6 Verify Full Stack Through agentgateway

```bash
GATEWAY_IP=$(kubectl get gateway amss-gateway -n agentgateway-system \
  -o jsonpath='{.status.addresses[0].value}')

# BFF health
curl -s http://${GATEWAY_IP}/health
# {"data":{"status":"ok","service":"bff"},"error":null}

# Crew store
curl -s http://${GATEWAY_IP}/api/v1/crew | jq '.data.total'
# 20

# KB store
curl -s http://${GATEWAY_IP}/api/v1/kb | jq '.data.total'
# 30

# Ticket store
curl -s http://${GATEWAY_IP}/api/v1/tickets | jq '.data.total'
# 15

# Frontend (full UI — works end-to-end via gateway)
open http://${GATEWAY_IP}
```

### 3.7 Run Activity Generator via Gateway

```bash
cd activity-generator
GATEWAY_IP=$(kubectl get gateway amss-gateway -n agentgateway-system \
  -o jsonpath='{.status.addresses[0].value}')
BFF_URL=http://${GATEWAY_IP} go run .
```

Traffic flows through the agentgateway and is visible in the Solo Enterprise UI observability dashboard.

---

## Phase 4 — Solo Enterprise for kagent

Install kagent Enterprise to replace the BFF stub mode with live AI agents backed by real LLM calls.

### 4.1 Set Environment Variables

```bash
export KAGENT_ENT_VERSION=0.3.19
export MGMT_CONTEXT=$(kubectl config current-context)
export ANTHROPIC_API_KEY=<your-key>
export AGENTGATEWAY_LICENSE_KEY=<your-key>
```

### 4.2 Install kagent Enterprise CRDs

```bash
helm upgrade -i kagent-crds \
  oci://us-docker.pkg.dev/solo-public/kagent-enterprise-helm/charts/kagent-enterprise-crds \
  --kube-context ${MGMT_CONTEXT} \
  -n kagent --create-namespace \
  --version ${KAGENT_ENT_VERSION}
```

### 4.3 Install Management Chart in kagent Namespace

The management chart must be in the `kagent` namespace — the kagent controller expects `solo-enterprise-ui` in its own namespace for OIDC. If you previously installed it in `agentgateway-system`, uninstall it first (the CRDs are cluster-scoped and conflict if installed twice):

```bash
# Only if previously installed in agentgateway-system:
helm uninstall management -n agentgateway-system

helm upgrade -i kagent-mgmt \
  oci://us-docker.pkg.dev/solo-public/solo-enterprise-helm/charts/management \
  --namespace kagent \
  --version 0.3.19 \
  --set cluster="mgmt-cluster" \
  --set products.kagent.enabled=true \
  --set products.agentgateway.enabled=true \
  --set-string licensing.licenseKey=${AGENTGATEWAY_LICENSE_KEY}
```

Access the UI:
```bash
kubectl port-forward service/solo-enterprise-ui -n kagent 4000:80 &
open http://localhost:4000
```

### 4.4 Create JWT Secret

Required before installing the kagent chart:

```bash
openssl genrsa -out /tmp/key.pem 2048
kubectl create secret generic jwt \
  -n kagent \
  --from-file=jwt=/tmp/key.pem \
  --dry-run=client -o yaml | kubectl apply -f -
```

### 4.5 Install kagent with Anthropic Provider

Create the values file:

```bash
cat << EOF > kagent.yaml
licensing:
  licenseKey: ${AGENTGATEWAY_LICENSE_KEY}
providers:
  default: anthropic
  anthropic:
    apiKey: ${ANTHROPIC_API_KEY}
otel:
  tracing:
    enabled: true
    exporter:
      otlp:
        endpoint: solo-enterprise-telemetry-collector.kagent.svc.cluster.local:4317
        insecure: true
EOF
```

Install:

```bash
helm upgrade -i kagent \
  oci://us-docker.pkg.dev/solo-public/kagent-enterprise-helm/charts/kagent-enterprise \
  -n kagent \
  --version ${KAGENT_ENT_VERSION} \
  --values kagent.yaml
```

Verify all pods running:

```bash
kubectl get pods -n kagent
```

Expected pods: `kagent-controller`, `kagent-postgresql`, `kmcp-enterprise-controller-manager`, `kagent-mgmt-clickhouse`, `solo-enterprise-telemetry-collector`, `solo-enterprise-ui`.

### 4.6 Create ModelConfig

The `provider` field is case-sensitive — must be `Anthropic` not `anthropic`:

```bash
kubectl apply -f - <<EOF
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: default-model-config
  namespace: kagent
spec:
  provider: Anthropic
  model: claude-sonnet-4-6
EOF
```

### 4.7 Register MCP Servers as RemoteMCPServer CRDs

The `allowedNamespaces.from: All` field is required because agents live in the `kagent` namespace but the MCP servers are in `amss`:

```bash
kubectl apply -f - <<EOF
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: kb-mcp
  namespace: amss
spec:
  description: "Knowledge Base MCP server"
  url: http://kb-mcp.amss.svc.cluster.local:9001
  protocol: STREAMABLE_HTTP
  allowedNamespaces:
    from: All
EOF

kubectl apply -f - <<EOF
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: ticket-mcp
  namespace: amss
spec:
  description: "Ticket MCP server"
  url: http://ticket-mcp.amss.svc.cluster.local:9002
  protocol: STREAMABLE_HTTP
  allowedNamespaces:
    from: All
EOF
```

### 4.8 Deploy Agents

Agents live in the `kagent` namespace so they share the namespace with `default-model-config`. The agent YAMLs already have `namespace: kagent` and explicit `modelConfig: default-model-config`:

```bash
kubectl apply -f agents/mission-support-agent/agent.yaml
kubectl apply -f agents/kb-curator-agent/agent.yaml
```

Verify both agents reach Ready:

```bash
kubectl get agents -n kagent
```

Expected: both show `READY: True` and `ACCEPTED: True`.

### 4.9 Test Agent via A2A

```bash
kubectl port-forward svc/kagent-controller -n kagent 8083:8083 &

# Check agent card
curl -s http://localhost:8083/api/a2a/kagent/mission-support-agent/.well-known/agent.json | jq .

# Invoke the agent (trailing slash on URL is required; use "kind" not "type" in message parts)
curl --max-time 120 -X POST http://localhost:8083/api/a2a/kagent/mission-support-agent/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "id": "test-1",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"kind": "text", "text": "What is the WCS flush procedure?"}]
      }
    }
  }'
```

Expected: agent runs `search_kb`, reads KB-001, returns a formatted procedure response with panel locations and valve IDs. Full chain confirmed: user → kagent A2A → Claude Sonnet 4.6 → MCP tools → KB Store → response.

### 4.10 Remaining Steps (TODO)

- Wire BFF to kagent A2A endpoint, set `STUB_MODE=false`
- Configure agentgateway egress for LLM traffic (guardrails, model failover)
- End-to-end test from frontend through agentgateway to live agents

---

## Teardown

### Remove AMSS Application

```bash
./k8s/teardown.sh
```

Deletes the `amss` namespace and every resource inside it (deployments, services, pods, secrets).

### Remove Solo Enterprise Products

Uninstall in reverse order:

```bash
# kagent (Phase 4)
helm uninstall kagent -n kagent
helm uninstall kagent-crds -n kagent
helm uninstall kagent-mgmt -n kagent
kubectl delete namespace kagent

# agentgateway (Phase 3)
helm uninstall enterprise-agentgateway -n agentgateway-system
helm uninstall enterprise-agentgateway-crds -n agentgateway-system
kubectl delete namespace agentgateway-system
```

### Remove Cluster

```bash
# OrbStack: cluster persists across restarts — reset via OrbStack app UI
# kind:
kind delete cluster --name amss
```

---

## Troubleshooting

### Store tests fail with "seed file not found"

The `SEED_PATH` env var must point to the seed JSON relative to where you're running. When running from the service directory:

```bash
SEED_PATH=seed-data/kb.json go run .
```

### Docker build fails with "COPY with more than one source"

Ensure Dockerfiles don't reference `go.sum` if the module has no external dependencies. Fix: remove `go.sum` from the COPY line.

### OrbStack k8s not responding

```bash
orb status
orb restart k8s
```

### BFF returns 404 for /api/v1/* routes

The BFF handles both root-path routes (for local dev via Vite proxy) and `/api/v1/` routes (for k8s). The `/api/v1/kb` path is rewritten to `/articles` at the kb-store. Confirm with:
```bash
curl -s http://localhost:30080/api/v1/kb/KB-001 | jq '.data.title'
```

### Frontend can't reach BFF in k8s

The frontend makes relative `/api/v1/...` calls. These only work when frontend and BFF are behind the same host (agentgateway at `192.168.139.2`). Direct NodePort access at `localhost:30081` routes relative calls back to the frontend service, not the BFF.

Options:
- Use agentgateway (Phase 3) for end-to-end frontend access
- Use `npm run dev` locally (Vite proxy routes `/api/v1/*` to `localhost:8080`)
- Build with explicit API URL for NodePort-only access: `docker build --build-arg VITE_API_URL=http://localhost:30080 -t amss/frontend:latest ./frontend`

### Frontend fails with "crypto.randomUUID is not a function"

`crypto.randomUUID()` requires a secure context (HTTPS or localhost). When accessing the frontend via plain HTTP on a non-localhost address (e.g., through agentgateway on `192.168.139.2`), the browser disables the Web Crypto API. The frontend has a Math.random-based fallback UUID generator for this case — this is already fixed in the current code.

### Activity generator gets HTTP 400 on ticket creation

The ticket-store requires specific field names. Check that ticket create bodies in `scenarios.json` include:
- `title`, `description`, `severity`, `mission` — required strings
- `category` — required, must be one of: `life-support`, `navigation`, `comms`, `power`, `propulsion`, `eva`, `medical`, `operations`, `thermal`, `structures`
- `reported_by` — required, the crew ID of the person filing the ticket
- `assigned_to` — optional (not `assignee`)

Fields not accepted by the ticket store: `status` (always `open` on create), `tags` (not a ticket field).

### agentgateway HTTPRoutes not attaching

Check the ReferenceGrant is in place — without it the gateway cannot reference services in the `amss` namespace:

```bash
kubectl get referencegrant -n amss
kubectl describe httproute bff-api-route -n amss
kubectl describe httproute frontend-route -n amss
```

The `parentRef` on both routes must match `amss-gateway` in `agentgateway-system` exactly.

### kmcp deploy fails

Ensure the kmcp controller CRDs are installed:

```bash
kmcp install
kubectl get crd mcpservers.kagent.dev
```

### kagent controller CrashLoopBackOff — "solo-enterprise-ui not found"

The management chart must be installed in the `kagent` namespace, not `agentgateway-system`. The kagent controller looks for `solo-enterprise-ui` in its own namespace for OIDC. Fix: uninstall from `agentgateway-system` and reinstall in `kagent` (see section 4.3).

### "cross-namespace reference not allowed" on RemoteMCPServer

Agents in the `kagent` namespace cannot reference MCP servers in `amss` without explicit permission. Add `allowedNamespaces.from: All` to each `RemoteMCPServer` spec and re-apply.

### "ModelConfig not found" or agent stuck with empty model

The kagent controller does not apply a default `modelConfig` automatically. The agent YAML must include an explicit `modelConfig: default-model-config` under `spec.declarative`. If you removed it assuming it was a default, add it back and re-apply.

### Model 404 error from LLM provider

Check the `model` string in the `ModelConfig` exactly matches what the provider accepts (`claude-sonnet-4-6`, not `claude-sonnet-4-20250514` or similar aliases). Delete and re-apply the `ModelConfig`, then delete and re-apply the agents to pick up the change.

### A2A returns empty response or 307 redirect

Two common causes:
- **Missing trailing slash**: the A2A URL must end with `/` — `http://localhost:8083/api/a2a/kagent/mission-support-agent/` not without the slash
- **Wrong message part key**: use `"kind": "text"` not `"type": "text"` in the message parts array

---

## Version History

| Date | Phase | What Changed |
|---|---|---|
| 2026-05-09 | Phase 1 | Initial build. Three stores (225 tests), two MCP servers (106 tests, `kmcp init go --no-git`), BFF with stub mode (103 tests, 12 keyword patterns), React frontend, activity generator v1 (17 tests). |
| 2026-05-09 | Phase 2 | k8s manifests for all 7 services. `deploy.sh` + `teardown.sh`. Frontend behind NodePort 30081, BFF at NodePort 30080. Activity generator v2 rewrite: humanized 22-minute cycles, 6 narrative scenarios, concurrent goroutines, graceful SIGINT shutdown, `stash_as` carry mechanism for ticket IDs, context cancellation throughout. 451 → 480 total tests. |
| 2026-05-09 | Phase 3 | Solo Enterprise agentgateway installed. Gateway at `192.168.139.2`. HTTPRoutes: `/api/v1/*` → BFF, `/*` → frontend. ReferenceGrant for cross-namespace access. Solo Enterprise UI at localhost:4000 via port-forward. Frontend `VITE_API_URL` removed from build — SPA uses relative paths that work through the gateway. Fixed `crypto.randomUUID` fallback for plain-HTTP contexts. Fixed activity generator ticket create field names (`category`, `reported_by`, `assigned_to`). |
| 2026-05-10 | Phase 4 | kagent Enterprise installed in `kagent` namespace with Anthropic provider (claude-sonnet-4-6). ModelConfig, RemoteMCPServer CRDs, and Agent CRDs applied. Both agents (mission-support-agent, kb-curator-agent) show READY: True. Full A2A chain verified: user → kagent → Claude Sonnet 4.6 → MCP tools → KB Store. Management chart moved from `agentgateway-system` to `kagent` namespace. Key gotchas: management chart namespace, `allowedNamespaces.from: All` on RemoteMCPServer, explicit `modelConfig` in agent YAML, trailing slash on A2A URL, `"kind"` not `"type"` in message parts. Remaining: BFF wiring, agentgateway LLM egress, frontend end-to-end. |
