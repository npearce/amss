# AMSS — Kubernetes Deployment

Local k8s manifests for the AMSS application (Phase 2 — stores + MCP servers + BFF).
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
1. Builds all 6 container images (`amss/kb-store`, `amss/ticket-store`, `amss/crew-store`, `amss/kb-mcp`, `amss/ticket-mcp`, `amss/bff`)
2. Applies manifests in order (namespace → stores → MCP servers → BFF)
3. Waits for all deployments to reach `Ready`
4. Prints the access URL

**kind users**: uncomment the `kind load docker-image` lines in `deploy.sh` before running.

## Access the BFF

### OrbStack (NodePort — no extra steps)

```bash
curl http://localhost:30080/health
```

### Any cluster (port-forward)

```bash
kubectl port-forward svc/bff 8080:8080 -n amss
# In another terminal:
curl http://localhost:8080/health
```

## Verify Each Service

All commands assume port-forward to BFF is active on 8080.

### BFF health
```bash
curl http://localhost:8080/health
# {"data":{"status":"ok","service":"bff"},"error":null}
```

### KB store (via BFF proxy)
```bash
curl http://localhost:8080/articles | jq '.data.total'
# 30
curl http://localhost:8080/articles/KB-001 | jq '.data.title'
```

### Ticket store (via BFF proxy)
```bash
curl http://localhost:8080/tickets | jq '.data.total'
# 15
curl http://localhost:8080/tickets/AMSS-001 | jq '.data.title'
```

### Crew store (via BFF proxy)
```bash
curl http://localhost:8080/crew | jq '.data.total'
# 20
```

### Chat (stub mode — no agent required)
```bash
curl -s -X POST http://localhost:8080/chat \
  -H 'Content-Type: application/json' \
  -d '{"crew_id":"wiseman-r","session_id":"test-001","mission":"artemis-ii","message":"WCS pressure is dropping"}' \
  | jq '.data'
```

### KB curation (stub mode)
```bash
curl -s -X POST http://localhost:8080/curate \
  -H 'Content-Type: application/json' \
  -d '{}' \
  | jq '.data.duplicates_flagged'
```

### Reset all stores to seed data
```bash
curl -s -X POST http://localhost:8080/reset | jq '.data'
```

### Direct store access (port-forward to individual services)
```bash
# KB store
kubectl port-forward svc/kb-store 8081:8081 -n amss
curl http://localhost:8081/health

# Ticket store
kubectl port-forward svc/ticket-store 8082:8082 -n amss
curl http://localhost:8082/health

# Crew store
kubectl port-forward svc/crew-store 8083:8083 -n amss
curl http://localhost:8083/health
```

## Deployment Notes

### Image pull policy
All deployments use `imagePullPolicy: Never` — images must be built locally before deploying. The deploy script handles this.

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
