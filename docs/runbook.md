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

## Phase 4 — Solo Enterprise for kagent (Pending License Key)

Install kagent Enterprise to replace the BFF stub mode with live AI agents backed by real LLM calls routed through agentgateway egress.

### 4.1 Install kagent Enterprise

Follow: https://docs.solo.io/kagent-enterprise/docs/latest/quickstart/

Key steps:
1. Install Solo distribution of Istio in ambient mode
2. Install Solo Enterprise for kagent via Helm (includes ambient mesh enrollment)
3. Configure Anthropic as the LLM provider in the model config
4. Verify all kagent pods running:

```bash
kubectl get pods -n kagent
```

### 4.2 Register MCP Servers with kmcp

Deploy the MCP servers as `MCPServer` CRDs managed by the kmcp controller:

```bash
cd mcp-servers/kb-mcp
kmcp deploy

cd ../ticket-mcp
kmcp deploy
```

Verify:
```bash
kubectl get mcpservers -n amss
```

### 4.3 Apply Agent CRDs

```bash
kubectl apply -f agents/mission-support-agent/agent.yaml -n amss
kubectl apply -f agents/kb-curator-agent/agent.yaml -n amss
```

Verify agents reach Ready state:
```bash
kubectl get agents -n amss
```

### 4.4 Configure agentgateway Egress for LLM Traffic

Configure guardrails, model failover, and content-based routing for agent → LLM traffic through agentgateway egress. Follow: https://docs.solo.io/agentgateway/2.3.x/

### 4.5 Flip BFF Out of Stub Mode

Once agents are deployed and reachable:
```bash
kubectl set env deployment/bff STUB_MODE=false -n amss
kubectl rollout status deployment/bff -n amss
```

### 4.6 Verify Live Agent Responses

```bash
GATEWAY_IP=$(kubectl get gateway amss-gateway -n agentgateway-system \
  -o jsonpath='{.status.addresses[0].value}')

curl -s -X POST http://${GATEWAY_IP}/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"demo-1","mission":"artemis-ii","message":"What is the WCS flush procedure?"}' \
  | jq '.data.response'
```

Response now comes from the mission-support-agent via real LLM call, routed through agentgateway. Observe the call in the Solo Enterprise UI.

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
helm uninstall kagent-enterprise -n kagent

# Management UI
helm uninstall management -n agentgateway-system

# agentgateway
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

---

## Version History

| Date | Phase | What Changed |
|---|---|---|
| 2026-05-09 | Phase 1 | Initial build. Three stores (225 tests), two MCP servers (106 tests, `kmcp init go --no-git`), BFF with stub mode (103 tests, 12 keyword patterns), React frontend, activity generator v1 (17 tests). |
| 2026-05-09 | Phase 2 | k8s manifests for all 7 services. `deploy.sh` + `teardown.sh`. Frontend behind NodePort 30081, BFF at NodePort 30080. Activity generator v2 rewrite: humanized 22-minute cycles, 6 narrative scenarios, concurrent goroutines, graceful SIGINT shutdown, `stash_as` carry mechanism for ticket IDs, context cancellation throughout. 451 → 480 total tests. |
| 2026-05-09 | Phase 3 | Solo Enterprise agentgateway installed. Gateway at `192.168.139.2`. HTTPRoutes: `/api/v1/*` → BFF, `/*` → frontend. ReferenceGrant for cross-namespace access. Solo Enterprise UI at localhost:4000 via port-forward. Frontend `VITE_API_URL` removed from build — SPA uses relative paths that work through the gateway. Fixed `crypto.randomUUID` fallback for plain-HTTP contexts. Fixed activity generator ticket create field names (`category`, `reported_by`, `assigned_to`). |
| — | Phase 4 | Pending: kagent Enterprise, Agent CRDs, agentgateway LLM egress, STUB_MODE=false. |
