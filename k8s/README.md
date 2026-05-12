# AMSS — Kubernetes Deployment

Local k8s manifests for the AMSS application (Phase 2 — stores + MCP servers + BFF + frontend + agentgateway routes).
Agents come in Phase 3 via kagent Agent CRDs.

## Prerequisites

- [OrbStack](https://orbstack.dev) (preferred) or [kind](https://kind.sigs.k8s.io)
- Docker (for building images)
- `kubectl` configured to point at your local cluster

Verify your cluster is up:
```bash
kubectl get nodes
```

## Deploy

Run from the **repo root**:

```bash
./k8s/deploy.sh
```

The script:
1. Builds all 7 container images (`amss/kb-store`, `amss/ticket-store`, `amss/crew-store`, `amss/kb-mcp`, `amss/ticket-mcp`, `amss/bff`, `amss/frontend`)
2. Applies manifests in order (namespace → stores → MCP servers → BFF → frontend → agentgateway routes)
3. Waits for all deployments to reach `Ready`
4. Prints the access URLs

**kind users**: uncomment the `kind load docker-image` lines in `deploy.sh` before running.

## Access the Application

### Via agentgateway port-forward (primary)

On OrbStack the agentgateway proxy LoadBalancer service does not receive an external IP. Port-forwarding is the standard local access method — the HTTPRoutes, ReferenceGrant, and backend selection all apply identically to a production LoadBalancer.

```bash
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
open http://localhost:8080
```

| | URL |
|---|---|
| Frontend + API | http://localhost:8080/ |
| BFF API | http://localhost:8080/api/v1/ |

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/crew | jq '.data.total'
```

HTTPRoutes are in `k8s/agentgateway-routes.yaml`. Check attachment status:
```bash
kubectl get httproute -n amss
```

### Solo Enterprise UI

```bash
kubectl port-forward svc/solo-enterprise-ui -n kagent 4000:80 &
open http://localhost:4000
```

### NodePort (debug only — bypasses agentgateway)

NodePort access bypasses agentgateway and is for BFF-only debug verification. The frontend does **not** work end-to-end via NodePort — relative `/api/v1/...` calls from `localhost:30081` resolve against the frontend service, not the BFF.

| Service | URL |
|---|---|
| BFF API only | http://localhost:30080 |
| Frontend only | http://localhost:30081 |

```bash
curl http://localhost:30080/health
```

## Verify Each Service

All commands use the agentgateway port-forward at `http://localhost:8080`. Start it first:

```bash
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
```

Substitute `localhost:30080` for direct BFF NodePort access if needed for debugging.

### Frontend (via gateway)
```bash
open http://localhost:8080/
# Astronaut Chat at /chat, Ground Control at /ground-control
```

### BFF health
```bash
curl http://localhost:8080/health
# {"data":{"status":"ok","service":"bff"},"error":null}
```

### KB store (via BFF — /api/v1 prefix)
```bash
curl http://localhost:8080/api/v1/kb | jq '.data.total'
# 30
curl http://localhost:8080/api/v1/kb/KB-001 | jq '.data.title'
```

### Ticket store (via BFF)
```bash
curl http://localhost:8080/api/v1/tickets | jq '.data.total'
# 15
curl http://localhost:8080/api/v1/tickets/AMSS-001 | jq '.data.title'
```

### Crew store (via BFF)
```bash
curl http://localhost:8080/api/v1/crew | jq '.data.total'
# 20
```

### Chat (stub mode — no agent required)
```bash
curl -s -X POST http://localhost:8080/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"test-001","mission":"artemis-ii","message":"WCS pressure is dropping"}' \
  | jq '.data'
```

### KB curation (stub mode)
```bash
curl -s -X POST http://localhost:8080/api/v1/curator \
  -H 'Content-Type: application/json' \
  -d '{}' \
  | jq '.data.duplicates_flagged'
```

### Reset all stores to seed data
```bash
curl -s -X POST http://localhost:8080/api/v1/reset | jq '.data'
```

### Direct store access (port-forward to individual services)
```bash
kubectl port-forward svc/kb-store 8081:8081 -n amss
curl http://localhost:8081/health

kubectl port-forward svc/ticket-store 8082:8082 -n amss
curl http://localhost:8082/health

kubectl port-forward svc/crew-store 8083:8083 -n amss
curl http://localhost:8083/health
```

## Deployment Notes

### Image pull policy
All deployments use `imagePullPolicy: Never` — images must be built locally before deploying. The deploy script handles this.

### agentgateway HTTPRoutes

`k8s/agentgateway-routes.yaml` creates three resources in the `amss` namespace:

| Resource | Kind | Purpose |
|---|---|---|
| `agentgateway-to-amss` | ReferenceGrant | Allows the Gateway in `agentgateway-system` to reference Services in `amss` |
| `bff-api-route` | HTTPRoute | Routes `/api/v1/*` → bff:8080 (higher priority — longer prefix) |
| `frontend-route` | HTTPRoute | Routes `/*` → frontend:80 (catch-all) |

Both HTTPRoutes attach to `agentgateway-proxy` in `agentgateway-system` via `parentRefs`. Check attachment:
```bash
kubectl get httproute -n amss -o wide
```

### Frontend API URL

`VITE_API_URL` is **not set** in the production build. The SPA calls `/api/v1/...` as a relative path, which works when the frontend and BFF are behind the same host. The agentgateway port-forward at `localhost:8080` satisfies this — the gateway routes `/api/v1/*` to the BFF and `/*` to the frontend from the same origin.

To build for direct NodePort access without agentgateway (dev/debug):
```bash
docker build --build-arg VITE_API_URL=http://localhost:30080 -t amss/frontend:latest ./frontend
```

### STUB_MODE
The BFF runs with `STUB_MODE=true`. Chat and curator endpoints return canned responses that cite real KB article IDs. This will be flipped to `false` in Phase 3 when kagent agents are deployed.

### MCP servers
MCP servers run with `--http :9001/9002` to use streamable HTTP transport instead of the default stdio mode. This makes them reachable over the network from agents.

### Seed data
All stores are ephemeral — data lives in memory and resets to seed on pod restart or `POST /reset`. This is by design.

## Tear Down

```bash
./k8s/teardown.sh
```

This deletes the `amss` namespace and all resources within it.

## What's Not Here Yet (Phase 3)

- `agents/mission-support-agent/agent.yaml` — kagent Agent CRD (authored, not applied yet)
- `agents/kb-curator-agent/agent.yaml` — kagent Agent CRD (authored, not applied yet)
- agentgateway LLM egress routing, guardrails, and model failover config
- Istio ambient mesh enrollment (`kubectl label namespace amss istio.io/dataplane-mode=ambient`)
- Helm chart (`helm/amss/`) for production deployment
