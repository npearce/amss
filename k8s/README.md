# AMSS — Kubernetes Deployment

Local k8s manifests for the AMSS application (Phase 2 — stores + MCP servers + BFF + frontend).
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
2. Applies manifests in order (namespace → stores → MCP servers → BFF → frontend)
3. Waits for all deployments to reach `Ready`
4. Prints the access URLs

**kind users**: uncomment the `kind load docker-image` lines in `deploy.sh` before running.

## Access the Application

### OrbStack (NodePort — no extra steps)

| Service | URL |
|---|---|
| Frontend | http://localhost:30081 |
| BFF API | http://localhost:30080 |

```bash
# Frontend
open http://localhost:30081

# BFF health
curl http://localhost:30080/health
```

### Any cluster (port-forward)

```bash
kubectl port-forward svc/frontend 3000:80 -n amss &
kubectl port-forward svc/bff 8080:8080 -n amss &
# Frontend at http://localhost:3000
# BFF at http://localhost:8080
```

## Verify Each Service

All curl commands use the BFF NodePort directly. Substitute `localhost:8080` if using port-forward.

### Frontend
```bash
open http://localhost:30081
# Astronaut Chat at /chat, Ground Control at /ground-control
```

### BFF health
```bash
curl http://localhost:30080/health
# {"data":{"status":"ok","service":"bff"},"error":null}
```

### KB store (via BFF — /api/v1 prefix)
```bash
curl http://localhost:30080/api/v1/kb | jq '.data.total'
# 30
curl http://localhost:30080/api/v1/kb/KB-001 | jq '.data.title'
```

### Ticket store (via BFF)
```bash
curl http://localhost:30080/api/v1/tickets | jq '.data.total'
# 15
curl http://localhost:30080/api/v1/tickets/AMSS-001 | jq '.data.title'
```

### Crew store (via BFF)
```bash
curl http://localhost:30080/api/v1/crew | jq '.data.total'
# 20
```

### Chat (stub mode — no agent required)
```bash
curl -s -X POST http://localhost:30080/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"test-001","mission":"artemis-ii","message":"WCS pressure is dropping"}' \
  | jq '.data'
```

### KB curation (stub mode)
```bash
curl -s -X POST http://localhost:30080/api/v1/curator \
  -H 'Content-Type: application/json' \
  -d '{}' \
  | jq '.data.duplicates_flagged'
```

### Reset all stores to seed data
```bash
curl -s -X POST http://localhost:30080/api/v1/reset | jq '.data'
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

### Frontend API URL

`VITE_API_URL=http://localhost:30080` is baked into the static build at image build time (Vite replaces it at bundle time). The SPA appends `/api/v1` internally, so all API calls go to `http://localhost:30080/api/v1/...`. Because the SPA runs in the user's browser, it must reach the BFF at a host-reachable address — cluster DNS (`bff.amss.svc.cluster.local`) is not accessible from the browser. The NodePort `localhost:30080` works for OrbStack and kind with port-forward.

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

- `agents/mission-support-agent/agent.yaml` — kagent Agent CRD
- `agents/kb-curator-agent/agent.yaml` — kagent Agent CRD
- agentgateway ingress and LLM routing config
- Istio ambient mesh enrollment (`kubectl label namespace amss istio.io/dataplane-mode=ambient`)
- Helm chart (`helm/amss/`) for production deployment
