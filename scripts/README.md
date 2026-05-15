# AMSS Scripts

Automation scripts for installing, running, and demoing the Artemis Mission Support System.

## Prerequisites

Three license keys from your Solo account representative, plus an Anthropic API key:

```bash
export AGENTGATEWAY_LICENSE_KEY=<key>   # agentgateway, kagent, management chart
export SOLO_ISTIO_LICENSE_KEY=<key>     # Solo distribution of Istio (ambient mesh)
export ANTHROPIC_API_KEY=<key>          # LLM provider
```

A running local k8s cluster (OrbStack preferred):

```bash
kubectl config use-context orbstack
kubectl get nodes
```

## Scripts

### setup.sh — One-command install from a clean cluster

```bash
./scripts/setup.sh
```

Installs the complete stack in order:
1. Gateway API CRDs
2. Solo Enterprise agentgateway (agentgateway-system namespace)
3. kagent management chart (kagent namespace)
4. kagent CRDs + controller with Anthropic provider + proxy.url routing
5. Solo distribution of Istio in ambient mode (istio-system namespace)
6. AMSS application (7 services — stores, MCP servers, BFF, frontend)
7. Ambient mesh enrollment for amss namespace
8. LLM egress config (AgentgatewayBackend, HTTPRoute for Anthropic)
9. Tracing policy (EnterpriseAgentgatewayPolicy + ReferenceGrant)
10. AMSS HTTPRoutes (agentgateway-routes.yaml)
11. ModelConfig, RemoteMCPServer CRDs, Agent CRDs
12. BFF switched to live agent mode (STUB_MODE=false)

**Estimated time: ~10 minutes** (Helm chart pulls dominate; AMSS image builds add ~3 minutes)

On completion, prints access URLs and demo track commands.

### teardown.sh — Complete removal

```bash
./scripts/teardown.sh
```

Removes everything in reverse install order: ambient mesh label → amss namespace → Istio → kagent → agentgateway → Gateway API CRDs.

Uses `|| true` on all uninstalls — safe to run even if only a subset of the stack was installed.

### demo-stop.sh — Stop a running demo

```bash
./scripts/demo-stop.sh
```

Stops the activity generator and all port-forwards without touching the cluster. Use this to cleanly end a demo between tracks or before handing off the machine. The lab environment (k8s deployments, Solo products) remains fully intact.

### demo-track1.sh — agentgateway only

**Audience**: Platform engineers, networking/security teams evaluating AI gateway capabilities.

**What it shows**: Inbound routing (frontend + BFF API), LLM egress traffic (token counts, latency, model routing), activity generator traffic flowing through the gateway.

**Requires**: Phases 2–3 (AMSS + agentgateway installed).

```bash
./scripts/demo-track1.sh
```

### demo-track2.sh — agentgateway + kagent

**Audience**: AI/ML engineers, platform teams evaluating agent orchestration.

**What it shows**: Real Claude Sonnet 4.6 responses via kagent A2A, declarative agent CRDs, MCP tool discovery via RemoteMCPServer, end-to-end traces (LLM call → tool invocation → KB search → response).

**Requires**: Phases 2–4 (AMSS + agentgateway + kagent installed).

```bash
./scripts/demo-track2.sh
```

### demo-track3.sh — agentgateway + ambient mesh

**Audience**: Platform/security teams evaluating zero-trust service mesh without sidecar overhead.

**What it shows**: All pods at 1/1 (no sidecars), ztunnel DaemonSet at node level, namespace label enrollment, mTLS on all amss east-west traffic visible in the Solo Enterprise UI service graph.

**Requires**: Phases 2–3, 5 (AMSS + agentgateway + ambient mesh installed; kagent optional).

```bash
./scripts/demo-track3.sh
```

### demo-track4.sh — Full Stack

**Audience**: Decision makers, full platform evaluations.

**What it shows**: The complete picture — agentgateway ingress + LLM egress, kagent declarative agents + MCP tools + traces, ambient mesh mTLS with zero app changes. Complete trace path from browser to KB Store and back.

**Requires**: Full stack (Phases 2–5, all products installed).

```bash
./scripts/demo-track4.sh
```

## Port-forwards (manual)

All demo scripts manage port-forwards automatically. To run them manually:

```bash
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
kubectl port-forward svc/solo-enterprise-ui -n kagent 4000:80 &
```

| URL | What |
|---|---|
| http://localhost:8080 | AMSS application (through agentgateway) |
| http://localhost:4000 | Solo Enterprise UI (kagent + agentgateway observability) |
