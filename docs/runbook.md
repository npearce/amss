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
export AGENTGATEWAY_LICENSE_KEY=<key>   # Used for agentgateway, kagent, and management chart
export SOLO_ISTIO_LICENSE_KEY=<key>     # Used for the Solo distribution of Istio (ambient mesh)
export ANTHROPIC_API_KEY=<key>          # LLM provider for agents
```

Store these securely. Do not commit them to the repo.

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

120 tests covering proxy routes (root-path and `/api/v1/` variants), stub chat and curator responses, CORS, reset, agent fallback behavior, and A2A client (`a2a.go`) — callAgent success/error/timeout/invalid-JSON, extractAgentText edge cases, extractKBReferences deduplication. Stub mode includes 12 keyword patterns that return realistic canned responses citing real KB article IDs.

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

  To access the application (through agentgateway):
    kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
    open http://localhost:8080

  To access the Solo Enterprise UI:
    kubectl port-forward svc/solo-enterprise-ui -n kagent 4000:80 &
    open http://localhost:4000

  Debug (direct NodePort, bypasses agentgateway — not recommended):
    BFF API only:  http://localhost:30080
    Frontend only: http://localhost:30081 (API calls won't work)
```

### 2.3 Verify

Primary access is through agentgateway via port-forward. Start it once and leave it running:

```bash
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
```

> **On OrbStack**, the agentgateway proxy LoadBalancer service does not get an external IP. Port-forwarding is the standard local access method and routes all traffic through agentgateway identically to a production LoadBalancer.

**BFF health:**
```bash
curl -s http://localhost:8080/health
# {"data":{"status":"ok","service":"bff"},"error":null}
```

**Crew store (20 members):**
```bash
curl -s http://localhost:8080/api/v1/crew | jq '.data.total'
# 20
```

**KB store (30 articles):**
```bash
curl -s http://localhost:8080/api/v1/kb | jq '.data.total'
# 30
```

**Ticket store (15 tickets):**
```bash
curl -s http://localhost:8080/api/v1/tickets | jq '.data.total'
# 15
```

**Chat (stub mode — no agent required):**
```bash
curl -s -X POST http://localhost:8080/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"test-001","mission":"artemis-ii","message":"WCS pressure is dropping"}' \
  | jq '.data'
```

Expected response references KB-001 (the WCS toilet pressure fault article).

**KB curation (stub mode):**
```bash
curl -s -X POST http://localhost:8080/api/v1/curator -H 'Content-Type: application/json' -d '{}' \
  | jq '.data.duplicates_flagged'
# 3
```

Returns 3 near-duplicate WCS articles flagged against KB-001.

**Reset all stores to seed data:**
```bash
curl -s -X POST http://localhost:8080/api/v1/reset | jq '.data'
```

**NodePort debug fallback** (bypasses agentgateway — BFF only, no frontend):
```bash
curl -s http://localhost:30080/health
```

> **NodePort note**: NodePort access is debug-only and bypasses agentgateway. The frontend at `localhost:30081` makes relative `/api/v1/...` calls that resolve against `localhost:30081` — the frontend service, not the BFF — so the UI won't work end-to-end. Use the port-forward approach above for full stack verification.

### 2.4 Run Activity Generator Against k8s

```bash
cd activity-generator
BFF_URL=http://localhost:8080 go run .
```

Requires the agentgateway port-forward to be running (`kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &`). One cycle fires 6 scenarios concurrently against the live BFF. After the cycle the stores reset and it repeats. Press Ctrl+C to stop (graceful shutdown).

Verify one cycle landed:
```bash
curl -s http://localhost:8080/api/v1/tickets | jq '.data.total'
# 18 (15 seed + 3 created by scenarios 2, 3, 6)
curl -s http://localhost:8080/api/v1/kb | jq '.data.total'
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
| bff | 120 |
| activity-generator | 46 |
| **Total** | **497** |

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
  name: agentgateway-proxy
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

> **Note**: The Solo Enterprise UI is installed in Phase 4 alongside kagent. agentgateway observability features require the management chart which is installed in the `kagent` namespace.

### 3.5 Configure LLM Egress through agentgateway

Routes agent LLM calls through agentgateway so all AI traffic is visible in the Solo Enterprise UI with observability, guardrails, and failover.

```bash
# 1. Create Anthropic API key secret for agentgateway
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: anthropic-secret
  namespace: agentgateway-system
type: Opaque
stringData:
  Authorization: ${ANTHROPIC_API_KEY}
EOF

# 2. Create AgentgatewayBackend — static TLS proxy to api.anthropic.com
# The agent sends its own API key in the Authorization header; agentgateway
# handles TLS origination and routes to the real Anthropic endpoint.
kubectl apply -f - <<EOF
apiVersion: agentgateway.dev/v1alpha1
kind: AgentgatewayBackend
metadata:
  name: anthropic
  namespace: agentgateway-system
spec:
  static:
    host: api.anthropic.com
    port: 443
  policies:
    tls: {}
EOF

# 3. Create HTTPRoute for LLM traffic — matches /v1/messages only
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: anthropic
  namespace: agentgateway-system
spec:
  parentRefs:
  - name: agentgateway-proxy
    namespace: agentgateway-system
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /v1/messages
    backendRefs:
    - name: anthropic
      namespace: agentgateway-system
      group: agentgateway.dev
      kind: AgentgatewayBackend
EOF

# 4. Configure tracing policy for agentgateway
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: agentgateway-to-telemetry
  namespace: kagent
spec:
  from:
  - group: enterpriseagentgateway.solo.io
    kind: EnterpriseAgentgatewayPolicy
    namespace: agentgateway-system
  to:
  - group: ""
    kind: Service
---
apiVersion: enterpriseagentgateway.solo.io/v1alpha1
kind: EnterpriseAgentgatewayPolicy
metadata:
  name: tracing
  namespace: agentgateway-system
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: Gateway
    name: agentgateway-proxy
  frontend:
    tracing:
      backendRef:
        name: solo-enterprise-telemetry-collector
        namespace: kagent
        kind: Service
        port: 4317
      randomSampling: "true"
EOF
```

Verify the backend and route are accepted:
```bash
kubectl get agentgatewaybackend -n agentgateway-system
kubectl get httproute anthropic -n agentgateway-system
```

### 3.6 Access the Application

```bash
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
open http://localhost:8080
```

On OrbStack the agentgateway proxy LoadBalancer service does not receive an external IP. Port-forwarding is the standard local access method and routes all traffic through agentgateway identically to a production LoadBalancer — the HTTPRoutes, ReferenceGrant, and backend selection all apply exactly the same way.

### 3.7 Verify Full Stack Through agentgateway

Port-forward must be running (see §3.6):

```bash
# BFF health
curl -s http://localhost:8080/health
# {"data":{"status":"ok","service":"bff"},"error":null}

# Crew store
curl -s http://localhost:8080/api/v1/crew | jq '.data.total'
# 20

# KB store
curl -s http://localhost:8080/api/v1/kb | jq '.data.total'
# 30

# Ticket store
curl -s http://localhost:8080/api/v1/tickets | jq '.data.total'
# 15

# Frontend (full UI — works end-to-end via gateway)
open http://localhost:8080
```

### 3.8 Run Activity Generator via Gateway

```bash
cd activity-generator
BFF_URL=http://localhost:8080 go run .
```

Traffic flows through the agentgateway port-forward and is visible in the Solo Enterprise UI observability dashboard.

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

The management chart must be in the `kagent` namespace — the kagent controller expects `solo-enterprise-ui` in its own namespace for OIDC:

```bash
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

> **Note**: `proxy.url` is intentionally omitted. That setting routes MCP connections through agentgateway, which breaks them. LLM egress is routed through the gateway via `anthropic.baseUrl` in the ModelConfig instead (see §4.7).

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

### 4.6 Access Solo Enterprise UIs

```bash
# AMSS application — through agentgateway
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
open http://localhost:8080

# Solo Enterprise UI — kagent + agentgateway observability dashboard
kubectl port-forward service/solo-enterprise-ui -n kagent 4000:80 &
open http://localhost:4000

# kagent A2A endpoint (debug — direct agent invocation without BFF)
kubectl port-forward svc/kagent-controller -n kagent 8083:8083 &
# http://localhost:8083/api/a2a/kagent/mission-support-agent/
```

### 4.7 Create ModelConfig

The `provider` field is case-sensitive (`Anthropic` not `anthropic`). The `anthropic.baseUrl` field routes LLM calls through the agentgateway proxy so all traffic is observable in the Solo Enterprise UI.

The kagent Helm chart creates its own `default-model-config` without `anthropic.baseUrl`. Delete it first to avoid a field ownership conflict, then apply the correct config:

```bash
kubectl delete modelconfig default-model-config -n kagent 2>/dev/null || true

kubectl apply -f - <<EOF
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: default-model-config
  namespace: kagent
spec:
  provider: Anthropic
  model: claude-sonnet-4-6
  apiKeySecret: kagent-anthropic
  apiKeySecretKey: ANTHROPIC_API_KEY
  anthropic:
    baseUrl: http://agentgateway-proxy.agentgateway-system.svc.cluster.local
EOF
```

`apiKeySecret` and `apiKeySecretKey` reference the secret created by the kagent Helm chart when `providers.anthropic.apiKey` is set in the values — the chart stores the key under `ANTHROPIC_API_KEY` in the `kagent-anthropic` secret. `anthropic.baseUrl` routes LLM calls through the agentgateway proxy for observability.

### 4.8 Register MCP Servers as RemoteMCPServer CRDs

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

### 4.9 Deploy Agents

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

### 4.10 Test Agent via A2A

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

Expected: agent runs `search_kb`, reads KB-001, returns a formatted procedure response with panel locations and valve IDs. Full chain confirmed: user → kagent A2A → Claude Sonnet 4.6 (via agentgateway egress) → MCP tools → KB Store → response.

After the request completes, open the Solo Enterprise UI (`http://localhost:4000`) and verify traces appear for both the kagent agent invocation and the agentgateway LLM egress call. Every tool execution and token exchange should be visible as spans.

### 4.11 Enable Live Agent Mode on BFF

```bash
kubectl set env deployment/bff -n amss \
  STUB_MODE=false \
  MISSION_SUPPORT_AGENT_URL=http://kagent-controller.kagent.svc.cluster.local:8083 \
  KAGENT_AGENT_NAMESPACE=kagent
```

Verify live agent response (port-forward must be running — see §3.6):

```bash
curl -s -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"crew_id":"wiseman-r","mission":"artemis-ii","session_id":"test-1","message":"What is the WCS flush procedure?"}' | jq '.data.response'
```

Expected: a detailed response citing KB-001 with panel locations and valve IDs — this is Claude Sonnet 4.6 responding via kagent, not a stub.

Then open the frontend at `http://localhost:8080` and test chat in the browser.

---

## Phase 5 — Ambient Mesh (Solo Distribution of Istio)

Install the Solo distribution of Istio in ambient mode to add mTLS and L7 observability to all east-west traffic between AMSS services — without sidecars. The ztunnel DaemonSet handles encryption at the node level; pods stay at 1/1 container throughout.

### Prerequisites

- `SOLO_ISTIO_LICENSE_KEY` — obtain from your Solo account representative
- Phases 2–4 complete (AMSS deployed, agentgateway and kagent installed)

### 5.1 Set Environment Variables

```bash
export SOLO_ISTIO_LICENSE_KEY=<your-key>
export ISTIO_VERSION=1.29.1
export ISTIO_IMAGE=${ISTIO_VERSION}-solo
export ISTIO_HUB=us-docker.pkg.dev/soloio-img/istio
export REPO=us-docker.pkg.dev/soloio-img/istio-helm
```

### 5.2 Install Solo Distribution of Istio (Ambient Mode)

Four charts in order: base CRDs, control plane, CNI node agent, ztunnel DaemonSet.

```bash
# 1. Istio base — CRDs and cluster roles
helm upgrade -i istio-base \
  oci://${REPO}/base \
  --version ${ISTIO_VERSION} \
  -n istio-system --create-namespace

# 2. Istiod — control plane with ambient profile
helm upgrade -i istiod \
  oci://${REPO}/istiod \
  --version ${ISTIO_VERSION} \
  -n istio-system \
  --set profile=ambient \
  --set-string global.hub="${ISTIO_HUB}" \
  --set-string global.tag="${ISTIO_IMAGE}" \
  --set-string licenseKey=${SOLO_ISTIO_LICENSE_KEY} \
  --wait

# 3. Istio CNI — node-level network programming (no sidecars)
helm upgrade -i istio-cni \
  oci://${REPO}/cni \
  --version ${ISTIO_VERSION} \
  -n istio-system \
  --set profile=ambient \
  --set-string global.hub="${ISTIO_HUB}" \
  --set-string global.tag="${ISTIO_IMAGE}"

# 4. ztunnel — per-node L4 proxy DaemonSet (handles mTLS)
helm upgrade -i ztunnel \
  oci://${REPO}/ztunnel \
  --version ${ISTIO_VERSION} \
  -n istio-system \
  --set-string global.hub="${ISTIO_HUB}" \
  --set-string global.tag="${ISTIO_IMAGE}"
```

Verify control plane and node agents are running:

```bash
kubectl get pods -n istio-system
```

Expected: `istiod` running, `istio-cni-node` DaemonSet pod on each node, `ztunnel` DaemonSet pod on each node.

### 5.3 Enroll the amss Namespace

Only the `amss` namespace is enrolled. The `kagent` and `agentgateway-system` namespaces are intentionally excluded:

| Namespace | Enrolled | Reason |
|---|---|---|
| `amss` | ✅ Yes | Protects data layer: stores ↔ MCP servers ↔ BFF with mTLS |
| `kagent` | ❌ No | Agents need direct outbound HTTPS to `api.anthropic.com` — ztunnel intercepts and breaks LLM API connections |
| `agentgateway-system` | ❌ No | Gateway manages its own TLS for ingress and LLM egress |

```bash
kubectl label namespace amss istio.io/dataplane-mode=ambient
```

Verify the label is set:

```bash
kubectl get namespace amss --show-labels | grep istio
```

### 5.4 Verify

**Pods stay at 1/1 — no sidecars injected:**

```bash
kubectl get pods -n amss
```

All pods should remain `1/1 Running`. Ambient mode uses the ztunnel DaemonSet at the node level — no containers are added to application pods.

**Application still works through agentgateway:**

```bash
curl -s http://localhost:8080/health
# {"data":{"status":"ok","service":"bff"},"error":null}

curl -s http://localhost:8080/api/v1/crew | jq '.data.total'
# 20

curl -s -X POST http://localhost:8080/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"test-1","mission":"artemis-ii","message":"WCS pressure is dropping"}' \
  | jq '.data.response'
```

**mTLS is active — observe it in the Solo Enterprise UI:**

```bash
kubectl port-forward service/solo-enterprise-ui -n kagent 4000:80 &
open http://localhost:4000
```

Navigate to the mesh observability view to see the service graph with mTLS indicators. All `amss` namespace east-west traffic (BFF → stores, MCP servers → stores) is encrypted by ztunnel. Agent LLM calls from kagent travel outside the mesh directly to the Anthropic API.

> **Mesh scope**: The ambient mesh protects the data layer. `kagent` agents and `agentgateway-system` operate outside the mesh so they can make direct TLS connections to external LLM providers. mTLS-only posture (no `AuthorizationPolicy` resources) maintains allow-all behaviour — strong encryption without authorization restrictions that could break cross-namespace traffic.

---

## Demo Tracks

Four demo tracks using the same port-forwards: `kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &` and `kubectl port-forward service/solo-enterprise-ui -n kagent 4000:80 &`.

### Track 1 — agentgateway Only (Phases 2–3)

Focus: AI-native gateway, HTTPRoute-based routing, stub chat.

**What to show:**
1. `open http://localhost:8080` — full frontend loads, user switcher works
2. Select an astronaut, send a chat message — stub response cites KB articles
3. Solo Enterprise UI at `http://localhost:4000` — gateway observability dashboard, request counts, latency

**Curl demo:**
```bash
curl -s http://localhost:8080/api/v1/chat \
  -X POST -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"demo-1","mission":"artemis-ii","message":"WCS pressure is dropping"}' \
  | jq '.data'
```

### Track 2 — agentgateway + kagent (Phases 2–4)

Focus: Real AI agents, A2A protocol, tool use via MCP, end-to-end traces.

**What to show:**
1. Open Solo Enterprise UI `http://localhost:4000` — agents panel, both agents READY
2. Open frontend `http://localhost:8080`, send a chat — this hits kagent via A2A
3. Switch back to UI — show the trace: kagent invocation → LLM call (via agentgateway egress) → tool calls (search_kb, read_kb_article) → response
4. Discuss: every token, every tool call visible; agentgateway is the egress for LLM traffic

**Direct A2A curl (bypass BFF):**
```bash
kubectl port-forward svc/kagent-controller -n kagent 8083:8083 &
curl --max-time 120 -X POST http://localhost:8083/api/a2a/kagent/mission-support-agent/ \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"message/send","id":"demo-1","params":{"message":{"role":"user","parts":[{"kind":"text","text":"What is the WCS flush procedure?"}]}}}'
```

### Track 3 — agentgateway + Ambient Mesh (Phases 2–3, 5)

Focus: Zero-trust east-west mTLS without sidecars, service mesh observability.

**What to show:**
1. `kubectl get pods -n amss` — all 1/1, no sidecars
2. `kubectl get namespace amss --show-labels | grep istio` — `istio.io/dataplane-mode=ambient` set
3. Solo Enterprise UI — mesh observability view, service graph for `amss` namespace, mTLS badges on every edge
4. Make a request, watch the graph light up — BFF → kb-store, ticket-store, crew-store all mTLS encrypted

**Talking point**: "The data layer (stores, MCP servers, BFF) is encrypted with mTLS via ambient mesh. Agent traffic to LLM providers operates outside the mesh for compatibility — kagent needs direct HTTPS to the Anthropic API. The application code is completely unchanged. No certificate rotation, no sidecar lifecycle management. ztunnel handles it at the node level."

### Track 4 — Full Stack (Phases 2–5)

Full demo: all products, all traffic encrypted, all calls traced.

**Setup:**
```bash
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
kubectl port-forward service/solo-enterprise-ui -n kagent 4000:80 &
```

**Demo flow:**
1. Open frontend `http://localhost:8080` — switch to astronaut Reid Wiseman
2. Send chat: "WCS pressure is dropping, what do I do?" — real Claude Sonnet 4.6 response
3. Switch to Solo Enterprise UI `http://localhost:4000`:
   - **Gateway view**: inbound request trace (browser → agentgateway → BFF)
   - **Agents view**: kagent invocation span, tool calls, LLM egress calls
   - **Mesh view**: service graph with mTLS on all `amss` east-west edges (stores ↔ BFF ↔ MCP servers); `kagent` is outside the mesh so agent → LLM traffic is not shown here
4. Ask the curator: `curl -s -X POST http://localhost:8080/api/v1/curator -d '{}' -H 'Content-Type: application/json' | jq '.data'` — KB deduplication report
5. Run activity generator for 2 minutes: `BFF_URL=http://localhost:8080 go run ./activity-generator/.` — watch the service graph fill with traffic

---

## Teardown

### Remove Ambient Mesh (Phase 5 — if installed)

Remove the namespace label first, then uninstall in reverse install order:

```bash
# Remove namespace enrollment
kubectl label namespace amss istio.io/dataplane-mode-

# Uninstall Istio components (reverse order)
helm uninstall ztunnel -n istio-system
helm uninstall istio-cni -n istio-system
helm uninstall istiod -n istio-system
helm uninstall istio-base -n istio-system
kubectl delete namespace istio-system
```

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
# Via agentgateway port-forward (primary)
curl -s http://localhost:8080/api/v1/kb/KB-001 | jq '.data.title'
# Via NodePort (debug fallback)
curl -s http://localhost:30080/api/v1/kb/KB-001 | jq '.data.title'
```

### Frontend can't reach BFF in k8s

The frontend makes relative `/api/v1/...` calls. These only work when frontend and BFF are behind the same host. The agentgateway port-forward at `localhost:8080` satisfies this — both routes serve from the same origin so relative paths work. Direct NodePort access at `localhost:30081` routes calls back to the frontend service, not the BFF.

Options:
- Port-forward agentgateway (primary): `kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &`, then `open http://localhost:8080`
- Use `npm run dev` locally (Vite proxy routes `/api/v1/*` to `localhost:8080`)
- Build with explicit API URL for NodePort-only access: `docker build --build-arg VITE_API_URL=http://localhost:30080 -t amss/frontend:latest ./frontend`

### Frontend fails with "crypto.randomUUID is not a function"

`crypto.randomUUID()` requires a secure context (HTTPS or localhost). The standard port-forward access at `http://localhost:8080` is a secure context, so this does not apply in normal usage. If accessing the frontend via a non-localhost plain-HTTP address, the browser disables the Web Crypto API. The frontend has a Math.random-based fallback UUID generator for this edge case — this is already fixed in the current code.

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

The `parentRef` on both routes must match `agentgateway-proxy` in `agentgateway-system` exactly.

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

### ModelConfig Helm conflict on upgrade

The kagent Helm chart creates a `default-model-config` without `anthropic.baseUrl`. If you apply our config first, Helm will refuse to upgrade with a field ownership conflict. The fix is to delete the resource and let the next `kubectl apply` recreate it:

```bash
kubectl delete modelconfig default-model-config -n kagent
kubectl apply -f - <<EOF
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: default-model-config
  namespace: kagent
spec:
  provider: Anthropic
  model: claude-sonnet-4-6
  apiKeySecret: kagent-anthropic
  apiKeySecretKey: ANTHROPIC_API_KEY
  anthropic:
    baseUrl: http://agentgateway-proxy.agentgateway-system.svc.cluster.local
EOF
```

### agentgateway proxy LoadBalancer stuck in Pending on OrbStack

OrbStack does not provision external IPs for LoadBalancer services. Use `kubectl port-forward` instead — this routes traffic identically to a LoadBalancer, with the same HTTPRoutes and backend selection applying:

```bash
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
```

### No traces visible in Solo Enterprise UI

Check in order:
1. `EnterpriseAgentgatewayPolicy` tracing resource exists: `kubectl get enterpriseagentgatewaypolicy -n agentgateway-system`
2. `ReferenceGrant` allows cross-namespace access to the telemetry collector: `kubectl get referencegrant agentgateway-to-telemetry -n kagent`
3. The KubernetesCluster CR is registered in the management UI — without cluster registration the UI has no context for the incoming traces

### Agent LLM calls not flowing through agentgateway

LLM egress is routed via `anthropic.baseUrl` in the `ModelConfig`, not via `proxy.url` in kagent.yaml (`proxy.url` routes MCP connections and breaks them — do not use it). Verify the ModelConfig has the correct baseUrl:

```bash
kubectl get modelconfig default-model-config -n kagent -o yaml | grep baseUrl
# Should show: baseUrl: http://agentgateway-proxy.agentgateway-system.svc.cluster.local
```

If missing, delete and re-apply:

```bash
kubectl delete modelconfig default-model-config -n kagent
kubectl apply -f - <<EOF
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: default-model-config
  namespace: kagent
spec:
  provider: Anthropic
  model: claude-sonnet-4-6
  apiKeySecret: kagent-anthropic
  apiKeySecretKey: ANTHROPIC_API_KEY
  anthropic:
    baseUrl: http://agentgateway-proxy.agentgateway-system.svc.cluster.local
EOF
```

Then restart the agents to pick up the change:

```bash
kubectl delete agents --all -n kagent
kubectl apply -f agents/mission-support-agent/agent.yaml
kubectl apply -f agents/kb-curator-agent/agent.yaml
```

### istio-cni-node pod stuck at 0/1 Ready

The `istio-cni-node` DaemonSet pod may stay `0/1` if the CNI plugin directory is not writable on the node. On OrbStack this typically resolves within 60 seconds as OrbStack provisions the node directory. Check:

```bash
kubectl describe pod -l k8s-app=istio-cni-node -n istio-system
```

If the pod reports `FailedMount` or a permission error on `/opt/cni/bin`, restart the OrbStack k8s node via the OrbStack app UI (Kubernetes → Restart). The CNI DaemonSet will retry on node restart.

### Application stops working after labeling the amss namespace

If pods restart and fail after `kubectl label namespace amss istio.io/dataplane-mode=ambient`, ztunnel may not be ready yet. Check:

```bash
kubectl get pods -n istio-system -l app=ztunnel
kubectl get pods -n amss
```

If ztunnel pods are not Running, wait for them before labeling. If pods in `amss` are crashing, check that ztunnel has enrolled them:

```bash
kubectl logs -l app=ztunnel -n istio-system --tail=20
```

Remove the label, let pods stabilize, then re-apply once ztunnel is fully ready:

```bash
kubectl label namespace amss istio.io/dataplane-mode-
# wait for ztunnel to be ready
kubectl label namespace amss istio.io/dataplane-mode=ambient
```

### Agent LLM calls timeout after enrolling kagent in the mesh

If you label the `kagent` namespace with `istio.io/dataplane-mode=ambient`, ztunnel intercepts the agent's outbound HTTPS connections to `api.anthropic.com` and the calls fail with a timeout. Only `amss` should be enrolled. Remove the label to restore agent functionality:

```bash
kubectl label namespace kagent istio.io/dataplane-mode-
```

Wait for the kagent-controller to restart, then verify agents are working again:

```bash
kubectl rollout restart deployment/kagent-controller -n kagent
kubectl get agents -n kagent
```

### AuthorizationPolicy breaks cross-namespace traffic

If you create any `AuthorizationPolicy` with `action: ALLOW` in the `amss` namespace, Istio switches to deny-by-default for that namespace. All traffic not explicitly covered by an ALLOW policy will be rejected — including cross-namespace calls from kagent agents to MCP servers and BFF to stores.

To avoid this, do not create any `AuthorizationPolicy` resources in the `amss` namespace. The mTLS-only posture (no authorization policies) provides strong encryption with allow-all behaviour, which is the correct configuration for this demo.

If you accidentally created an `AuthorizationPolicy`, delete it:

```bash
kubectl delete authorizationpolicy --all -n amss
```

### Ambient mesh shows traffic but mTLS is not confirmed

In the Solo Enterprise UI mesh view, mTLS indicators appear only after traffic flows through ztunnel. Send a few requests to generate telemetry:

```bash
for i in $(seq 1 5); do curl -s http://localhost:8080/api/v1/crew | jq '.data.total'; done
```

Then refresh the Solo Enterprise UI. If mTLS still does not appear, verify ztunnel is running on the same node as the `amss` pods:

```bash
kubectl get pods -n istio-system -o wide | grep ztunnel
kubectl get pods -n amss -o wide
```

The ztunnel pod and the amss pods must share the same node for mTLS to be active on their traffic.

---

## Version History

| Date | Phase | What Changed |
|---|---|---|
| 2026-05-09 | Phase 1 | Initial build. Three stores (225 tests), two MCP servers (106 tests, `kmcp init go --no-git`), BFF with stub mode (103 tests, 12 keyword patterns), React frontend, activity generator v1 (17 tests). |
| 2026-05-09 | Phase 2 | k8s manifests for all 7 services. `deploy.sh` + `teardown.sh`. Frontend behind NodePort 30081, BFF at NodePort 30080. Activity generator v2 rewrite: humanized 22-minute cycles, 6 narrative scenarios, concurrent goroutines, graceful SIGINT shutdown, `stash_as` carry mechanism for ticket IDs, context cancellation throughout. 451 → 480 total tests. |
| 2026-05-09 | Phase 3 | Solo Enterprise agentgateway installed. Gateway at `192.168.139.2`. HTTPRoutes: `/api/v1/*` → BFF, `/*` → frontend. ReferenceGrant for cross-namespace access. Solo Enterprise UI at localhost:4000 via port-forward. Frontend `VITE_API_URL` removed from build — SPA uses relative paths that work through the gateway. Fixed `crypto.randomUUID` fallback for plain-HTTP contexts. Fixed activity generator ticket create field names (`category`, `reported_by`, `assigned_to`). |
| 2026-05-10 | Phase 4 | kagent Enterprise installed in `kagent` namespace with Anthropic provider (claude-sonnet-4-6). ModelConfig, RemoteMCPServer CRDs, and Agent CRDs applied. Both agents (mission-support-agent, kb-curator-agent) show READY: True. Full A2A chain verified: user → kagent → Claude Sonnet 4.6 → MCP tools → KB Store. Management chart moved from `agentgateway-system` to `kagent` namespace. Key gotchas: management chart namespace, `allowedNamespaces.from: All` on RemoteMCPServer, explicit `modelConfig` in agent YAML, trailing slash on A2A URL, `"kind"` not `"type"` in message parts. Remaining: BFF wiring, agentgateway LLM egress, frontend end-to-end. |
| 2026-05-11 | Phase 4 | BFF wired to kagent A2A endpoint (`a2a.go`, 17 new tests, 480 → 497 total). `STUB_MODE=false` enables real LLM responses. Full end-to-end chain verified: frontend → agentgateway → BFF → kagent A2A → Claude Sonnet 4.6 → MCP tools → KB Store. Runbook cleanup: consolidated management chart install to `kagent` namespace only (removed Phase 3 UI install step), added Access the Application section to Phase 3, added Access Solo Enterprise UIs section to Phase 4, replaced TODO list with completed BFF wiring steps. License keys simplified to two vars: `AGENTGATEWAY_LICENSE_KEY` and `ANTHROPIC_API_KEY`. |
| 2026-05-12 | Phase 3 | agentgateway LLM egress fully configured: `AgentgatewayBackend` for Anthropic, `HTTPRoute` for LLM traffic, `EnterpriseAgentgatewayPolicy` tracing with OTel collector. `proxy.url` added to `kagent.yaml` so all declarative agent LLM calls route through agentgateway. Both inbound (browser→BFF) and outbound (agent→LLM) traffic now flows through agentgateway with full observability in Solo Enterprise UI — agent invocations, LLM calls, tool executions all visible as traces. Port-forward established as the standard local access method (OrbStack does not provision LoadBalancer IPs). Troubleshooting entries added: ModelConfig Helm conflict, LoadBalancer Pending, missing traces, agent LLM bypass. |
| 2026-05-14 | Phase 3/4 | LLM egress corrected: `AgentgatewayBackend` changed from AI backend to static TLS proxy (`spec.static.host: api.anthropic.com`, `spec.policies.tls: {}`). `ModelConfig` updated to include `anthropic.baseUrl` pointing at the agentgateway proxy — this is what actually routes LLM calls through the gateway. `proxy.url` removed from `kagent.yaml` (it routes MCP connections, not LLM calls, and breaks MCP). HTTPRoute for Anthropic scoped to `/v1/messages` path prefix (catch-all broke frontend routing). ModelConfig delete-before-apply pattern adopted to avoid Helm field ownership conflicts. Troubleshooting entries updated. |
| 2026-05-12 | Phase 5 | Solo distribution of Istio 1.29.1 installed in ambient mode (no sidecars). Four Helm charts: `istio-base`, `istiod`, `istio-cni`, `ztunnel`. `amss` namespace labeled `istio.io/dataplane-mode=ambient` — all east-west traffic (BFF→stores, MCP→stores) now mTLS-encrypted by ztunnel at node level. App pods remain 1/1. mTLS confirmed in Solo Enterprise UI mesh view. Four demo tracks documented covering every product combination from agentgateway-only through full stack. Prerequisites updated (`SOLO_ISTIO_LICENSE_KEY`), teardown updated with Istio uninstall steps, troubleshooting entries added: istio-cni-node NotReady, post-label pod crashes, mTLS not visible in UI. |
