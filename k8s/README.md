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

### Via agentgateway (primary — Phase 3+)

Solo Enterprise agentgateway is the primary access point once installed.
Both frontend and BFF sit behind the same gateway address.

| | URL |
|---|---|
| Frontend | http://192.168.139.2/ |
| BFF API | http://192.168.139.2/api/v1/ |

```bash
open http://192.168.139.2/

curl http://192.168.139.2/health
curl http://192.168.139.2/api/v1/crew | jq '.data.total'
```

Because both the frontend and BFF are behind the same host, the frontend SPA calls `/api/v1/...` as a **relative path** — no `VITE_API_URL` is baked into the image. The agentgateway routes `/api/v1/*` to the BFF and everything else to the frontend.

HTTPRoutes are in `k8s/agentgateway-routes.yaml`. Check attachment status:
```bash
kubectl get httproute -n amss
```

### OrbStack NodePort (direct — dev/debug only)

| Service | URL |
|---|---|
| Frontend | http://localhost:30081 |
| BFF API | http://localhost:30080 |

```bash
open http://localhost:30081
curl http://localhost:30080/health
```

Note: when accessing the frontend via NodePort directly (not through agentgateway), the SPA makes relative `/api/v1/...` calls which resolve against `localhost:30081` — the frontend service, not the BFF. Use agentgateway for a working end-to-end flow, or run the frontend in dev mode with `npm run dev` (Vite proxy handles the routing).

### Any cluster (port-forward)

```bash
kubectl port-forward svc/frontend 3000:80 -n amss &
kubectl port-forward svc/bff 8080:8080 -n amss &
# Frontend at http://localhost:3000
# BFF at http://localhost:8080
```

## Verify Each Service

Commands use the agentgateway address `192.168.139.2`. Substitute `localhost:30080` for direct BFF NodePort access.

### Frontend (via gateway)
```bash
open http://192.168.139.2/
# Astronaut Chat at /chat, Ground Control at /ground-control
```

### BFF health
```bash
curl http://192.168.139.2/health
# {"data":{"status":"ok","service":"bff"},"error":null}
```

### KB store (via BFF — /api/v1 prefix)
```bash
curl http://192.168.139.2/api/v1/kb | jq '.data.total'
# 30
curl http://192.168.139.2/api/v1/kb/KB-001 | jq '.data.title'
```

### Ticket store (via BFF)
```bash
curl http://192.168.139.2/api/v1/tickets | jq '.data.total'
# 15
curl http://192.168.139.2/api/v1/tickets/AMSS-001 | jq '.data.title'
```

### Crew store (via BFF)
```bash
curl http://192.168.139.2/api/v1/crew | jq '.data.total'
# 20
```

### Chat (stub mode — no agent required)
```bash
curl -s -X POST http://192.168.139.2/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"test-001","mission":"artemis-ii","message":"WCS pressure is dropping"}' \
  | jq '.data'
```

### KB curation (stub mode)
```bash
curl -s -X POST http://192.168.139.2/api/v1/curator \
  -H 'Content-Type: application/json' \
  -d '{}' \
  | jq '.data.duplicates_flagged'
```

### Reset all stores to seed data
```bash
curl -s -X POST http://192.168.139.2/api/v1/reset | jq '.data'
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

Both HTTPRoutes attach to `amss-gateway` in `agentgateway-system` via `parentRefs`. Check attachment:
```bash
kubectl get httproute -n amss -o wide
```

### Frontend API URL

`VITE_API_URL` is **not set** in the production build. The SPA calls `/api/v1/...` as a relative path, which works when the frontend and BFF are behind the same gateway host (`192.168.139.2`). The gateway routes `/api/v1/*` to the BFF and `/*` to the frontend.

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
