# AMSS Scripts

## Scripts

| Script | What it does |
|---|---|
| `setup.sh` | Full stack install (agentgateway + kagent + Keycloak + AMSS app) |
| `setup.sh --with-mesh` | Above + Solo distribution of Istio ambient mesh |
| `teardown.sh` | Complete removal of all components |
| `demo.sh` | Start the demo (port-forwards + activity generator) |
| `demo-stop.sh` | Stop the demo without tearing down the environment |

Keycloak is deployed automatically as part of `setup.sh` — no separate step required.

## Quick Start

```bash
export AGENTGATEWAY_LICENSE_KEY=<key>
export ANTHROPIC_API_KEY=<key>
./scripts/setup.sh
./scripts/demo.sh
```

## With Ambient Mesh

```bash
export AGENTGATEWAY_LICENSE_KEY=<key>
export ANTHROPIC_API_KEY=<key>
export SOLO_ISTIO_LICENSE_KEY=<key>
./scripts/setup.sh --with-mesh
./scripts/demo.sh
```

`SOLO_ISTIO_LICENSE_KEY` is only required when using `--with-mesh`.

## Port-forwards (manual)

`demo.sh` manages port-forwards automatically. To run them manually:

```bash
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
kubectl port-forward svc/solo-enterprise-ui -n kagent 4000:80 &
```

| URL | What |
|---|---|
| http://localhost:8080 | AMSS application (through agentgateway) |
| http://localhost:4000 | Solo Enterprise UI (kagent + agentgateway observability) |
