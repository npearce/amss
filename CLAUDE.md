# AMSS — Artemis Mission Support System

## What This Is

A demonstration application showcasing Solo.io's commercial AI platform products through a fictional NASA Artemis mission support system. Astronauts and ground control use AI agents to query a knowledge base, manage support tickets, and get real-time help with spacecraft systems.

### Solo.io Commercial Products Used

This demo uses the **commercial/enterprise** editions, not the open source versions. License keys are required.

| Commercial Product | What It Does | License Key Env Var | Docs |
|---|---|---|---|
| **Solo Enterprise for kagent** | Agent runtime, CRDs, management UI, observability, multi-framework support | (part of install) | [docs.solo.io/kagent-enterprise](https://docs.solo.io/kagent-enterprise/docs/latest/) |
| **Solo Enterprise for agentgateway** | AI-native gateway for LLM traffic, MCP, guardrails, failover, observability | `AGENTGATEWAY_LICENSE_KEY` | [docs.solo.io/agentgateway](https://docs.solo.io/agentgateway/2.3.x/) |
| **Solo distribution of Istio** (ambient mode) | Service mesh — mTLS, L7 observability, policy enforcement, no sidecars | `SOLO_ISTIO_LICENSE_KEY` | via kagent-enterprise install |

In Solo Enterprise for kagent, agentgateway is installed as a waypoint proxy in the ambient mesh. They are tightly integrated — not separate installs.

## Ground Rules

### Language

Go for all backend services. Use the official Go MCP SDK (`github.com/modelcontextprotocol/go-sdk/mcp`) for MCP servers. Use Go standard library `net/http` (Go 1.22+ routing patterns) for REST APIs — no web framework dependencies. React + Vite for the frontend.

### Module Structure

Every service is its own Go module with its own `go.mod`. No `go.work`, no shared modules, no monorepo build. You `cd` into a service directory to build and test it. Module path prefix: `github.com/npearce/amss/<path-to-service>`.

Each service directory contains:
- `go.mod` — independent module
- `main.go` — entrypoint
- `*_test.go` — tests alongside the code they test
- `Dockerfile` — multi-stage build producing a minimal image
- `README.md` — build, test, run (local, container, k8s) instructions

### Testing

Every API endpoint gets tests for both success and error paths. Use Go standard `testing` package with `net/http/httptest`. Table-driven tests where it makes sense. Store tests use temporary in-memory state — no test pollution between runs. MCP server tests mock the HTTP client that talks to stores. BFF tests mock all downstream HTTP clients.

Everything must build and the test surface must be complete:
- `go build ./...` must succeed in every service directory
- `go test ./...` must pass in every service directory
- `go vet ./...` must be clean

### Data Stores (KB, Ticket, Crew)

Each store is a Go HTTP server backed by an in-memory data store:
- On startup, read the seed JSON file into memory
- Maintain maps for O(1) lookup by ID
- Build a simple inverted index on startup for text search (lowercase tokenized words → set of document IDs)
- Search does OR across tokens, ranked by match count
- Filter + search combo: filters narrow first, then search within that filtered set
- All CRUD operates on the in-memory structures
- No persistence to disk — the store is ephemeral; seed data is the restore point
- `/reset` endpoint reloads seed JSON and rebuilds the index

### API Envelope

All REST responses use a standard envelope:

```json
{"data": ..., "error": null}
```

On error:

```json
{"data": null, "error": {"code": "NOT_FOUND", "message": "Ticket AMSS-999 not found"}}
```

Each service defines its own envelope type. No shared types package.

### Frontend

React + Vite. No RBAC. Simple user switcher — a dropdown or sidebar to pick a crew member identity from the crew store. That identity is sent as `crew_id` on all API calls. Switching users is instant, no login flow.

Two views:
- **Astronaut Chat** — chat interface for the selected crew member
- **Ground Control** — KB management, ticket queue, crew activity, curator reports

### Activity Generator

Placeholder scenarios for v1. Each scenario is a JSON entry describing the API calls to make:
- Create a ticket
- Close a ticket
- Create a KB article
- Archive a KB article

The generator reads `scenarios.json` and fires them on a loop against the BFF API. Creative narrative comes later once we have an alpha build.

### Agents

Agents are **not Go services** — they are kagent `Agent` CRDs with `type: Declarative`. The kagent controller and Python ADK runtime handle LLM calls, tool orchestration, memory, and context compaction. We define agents as YAML manifests with system prompts and tool references to our MCP servers.

Two agents:
- **Mission Support Agent** — answers crew questions, searches KB, creates tickets
- **KB Curator Agent** — de-duplicates articles, auto-tags, scores usefulness

For local dev (docker-compose, no k8s), the BFF provides a stub mode that returns keyword-matched responses without requiring kagent or an LLM. Set `STUB_MODE=true` (default when agents are unreachable).

Agent deployment requires Phase 3 (kagent Enterprise on k8s).

### MCP Servers

Scaffolded with `kmcp init go --no-git`. Uses the official Go MCP SDK (`github.com/modelcontextprotocol/go-sdk/mcp`). Each follows the kmcp project structure: `cmd/server/main.go` entrypoint, `internal/tools/` for tool definitions, `internal/client/` for store HTTP clients. Supports stdio and streamable HTTP transports.

Two servers:
- **KB MCP** — search, read, create, update KB articles, list categories
- **Ticket MCP** — search, read, create, update tickets, add comments, get summary

### Containers and Deployment

Each Go service has a Dockerfile with multi-stage build (build stage + scratch/distroless runtime). Each service README documents:
- How to build: `go build -o <name> .`
- How to test: `go test ./... -v`
- How to run locally: `go run . [env vars]`
- How to build container: `docker build -t amss/<name>:latest .`
- How to deploy to k8s: raw manifests or helm chart

**k8s is the default deployment target.** OrbStack (preferred) or kind provide a local cluster. The workflow is: build → test → containerize → deploy to k8s. No docker-compose layer.

### Solo Enterprise Products (on the same k8s cluster)

- **Solo Enterprise for kagent**: Install via Helm per [quickstart](https://docs.solo.io/kagent-enterprise/docs/latest/quickstart/). Includes ambient mesh, agentgateway as waypoint, management UI, OTel, ClickHouse.
- **Solo Enterprise for agentgateway**: Install via Helm per [install guide](https://docs.solo.io/agentgateway/2.3.x/install/helm). Provides LLM gateway with guardrails, failover, content routing.
- License keys: `SOLO_ISTIO_LICENSE_KEY`, `GLOO_GATEWAY_LICENSE_KEY`, `AGENTGATEWAY_LICENSE_KEY`

---

## Architecture

### Traffic Flow

```
Activity Generator / UIs
        │
        ▼
  Solo Enterprise for agentgateway (ingress, port-forward :8080)
        │
        ▼
      BFF API ──────────────────────┐
        │                           │
        ▼                           ▼
  Mission Support Agent      KB Curator Agent
  (kagent Agent CRD)         (kagent Agent CRD)
        │                           │
        ▼                           ▼
  Solo Enterprise for agentgateway (egress — LLM traffic)
        │                           │
        ▼                           ▼
   Anthropic API             Anthropic API
        │                           │
   ┌────┴────┐                 ┌────┴────┐
   ▼         ▼                 ▼         ▼
KB MCP   Ticket MCP         KB MCP   Ticket MCP
   │         │                 │         │
   ▼         ▼                 ▼         ▼
KB Store  Ticket Store       KB Store  Ticket Store

         Crew Store (BFF direct access)
```

All east-west traffic (agents ↔ MCP servers ↔ stores) runs on the Solo distribution of Istio in ambient mode — mTLS and L7 observability with no sidecars. agentgateway is deployed as a waypoint proxy in the mesh.

### Solo.io Product Demo Coverage

| Product | Demo Moment |
|---|---|
| Solo Enterprise for kagent | Both agents as `Agent` CRDs, declarative config, management UI, observability, tracing |
| Solo Enterprise for agentgateway | Ingress (user→BFF), Egress (agent→LLM), guardrails, model failover, content-based routing |
| Solo distribution of Istio (ambient) | mTLS + L7 observability on all east-west traffic, no sidecars, policy enforcement |

---

## Project Structure

```
amss/
├── CLAUDE.md                        # This document
├── SPEC.md                          # Data schemas, API contract, seed data spec
│
├── stores/
│   ├── kb-store/
│   │   ├── go.mod
│   │   ├── main.go
│   │   ├── handlers.go              # HTTP route handlers
│   │   ├── handlers_test.go         # HTTP-level tests (good + bad paths)
│   │   ├── store.go                 # In-memory store, index, data types
│   │   ├── store_test.go            # Store logic + index tests
│   │   ├── seed-data/
│   │   │   └── kb.json
│   │   ├── Dockerfile
│   │   └── README.md
│   │
│   ├── ticket-store/                # Same shape as kb-store
│   │   ├── go.mod
│   │   ├── main.go
│   │   ├── handlers.go
│   │   ├── handlers_test.go
│   │   ├── store.go
│   │   ├── store_test.go
│   │   ├── seed-data/
│   │   │   └── tickets.json
│   │   ├── Dockerfile
│   │   └── README.md
│   │
│   └── crew-store/                  # Same shape, plus conversation log
│       ├── go.mod
│       ├── main.go
│       ├── handlers.go
│       ├── handlers_test.go
│       ├── store.go
│       ├── store_test.go
│       ├── seed-data/
│       │   └── crew.json
│       ├── Dockerfile
│       └── README.md
│
├── mcp-servers/
│   ├── kb-mcp/                          # Scaffolded with kmcp
│   │   ├── go.mod
│   │   ├── cmd/server/main.go           # Entrypoint (stdio + HTTP transport)
│   │   ├── internal/client/client.go    # KB store HTTP client
│   │   ├── internal/client/client_test.go
│   │   ├── internal/tools/              # One file per MCP tool
│   │   │   ├── all_tools.go             # Tool registry
│   │   │   ├── search_kb.go
│   │   │   ├── read_kb_article.go
│   │   │   ├── create_kb_article.go
│   │   │   ├── update_kb_article.go
│   │   │   ├── list_kb_categories.go
│   │   │   └── tools_test.go
│   │   ├── kmcp.yaml                    # kmcp deployment config
│   │   ├── Dockerfile
│   │   └── README.md
│   │
│   └── ticket-mcp/                      # Same kmcp structure
│       ├── go.mod
│       ├── cmd/server/main.go
│       ├── internal/client/
│       ├── internal/tools/
│       ├── kmcp.yaml
│       ├── Dockerfile
│       └── README.md
│
├── agents/
│   ├── mission-support-agent/
│   │   ├── agent.yaml                   # kagent Agent CRD (Declarative)
│   │   └── prompts/
│   │       └── system.md                # System prompt (referenced in agent.yaml)
│   │
│   └── kb-curator-agent/
│       ├── agent.yaml                   # kagent Agent CRD (Declarative)
│       └── prompts/
│           └── system.md
│
├── bff/
│   ├── go.mod
│   ├── main.go
│   ├── handlers.go
│   ├── handlers_test.go
│   ├── config.go                    # Service URLs via env vars
│   ├── Dockerfile
│   └── README.md
│
├── frontend/
│   ├── package.json
│   ├── vite.config.js
│   ├── src/
│   │   ├── App.jsx
│   │   ├── views/
│   │   │   ├── AstronautChat.jsx
│   │   │   └── GroundControl.jsx
│   │   └── components/
│   ├── Dockerfile
│   └── README.md
│
├── activity-generator/
│   ├── go.mod
│   ├── main.go
│   ├── generator.go
│   ├── generator_test.go
│   ├── scenarios.json               # Placeholder: create/close ticket, create/archive KB
│   ├── Dockerfile
│   └── README.md
│
├── helm/
│   └── amss/
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
│
└── scripts/
    ├── build-all.sh                 # Iterates services, runs go build
    ├── test-all.sh                  # Iterates services, runs go test
    └── seed.sh                      # Reset all stores to seed data
```

---

## Seed Data Summary

### Crew (20 members)

- **Artemis II** (4 astronauts): Wiseman, Glover, Koch, Hansen (real crew)
- **Artemis III** (4 astronauts): Patel, Okafor, Chen, Bergström (fictional)
- **Artemis IV** (4 astronauts): Tanaka, Morales, Adeyemi, Kovács (fictional)
- **Ground Control** (8): Torres (Flight Director), McKinnon (CAPCOM), Vasquez (ECLSS), Ivanov (GNC), Webb (Systems), Nair (Flight Surgeon), Osei (COMM), Volkov (EVA)

### KB Articles (30)

10 categories: life-support, navigation, comms, power, propulsion, eva, medical, operations, thermal, structures. Intentional near-duplicates (KB-001/018/026/030 all cover WCS/toilet) for curator demo. Mix of well-tagged and empty-tagged articles.

### Tickets (15)

2×P1, 4×P2, 7×P3, 2×P4. Star ticket: AMSS-001 (WCS toilet pressure fault) with 5-comment resolution thread.

---

## Build Phases

### Phase 1 — Data Stores ✅
1. **Stores** (kb-store, ticket-store, crew-store) — in-memory JSON stores, full test coverage

### Phase 2 — Application Layer
2. **MCP servers** (kb-mcp, ticket-mcp) — scaffolded with `kmcp`, wrap store APIs as MCP tools ✅
3. **BFF** — proxies stores, stubs agent responses when kagent unavailable ✅
4. **Frontend** — React + Vite, user switcher, astronaut chat + ground control views ✅
5. **Activity generator** — 6 narrative scenarios, humanized 22-minute cycles, graceful shutdown ✅
6. **k8s manifests** — deploy stores, MCP servers, BFF, frontend to OrbStack/kind (agents stubbed) ✅

### Phase 3 — Solo Enterprise Integration (k8s)
7. **Solo Enterprise for kagent** — install on OrbStack, deploy Agent CRDs (declarative YAML) ✅
8. **Solo Enterprise for agentgateway** — ingress + LLM egress via `AgentgatewayBackend`, tracing policy ✅
9. **MCP server deployment** — `RemoteMCPServer` CRDs applied in `amss` namespace ✅
10. **Agent CRDs** — mission-support-agent and kb-curator-agent as `Agent` resources ✅
11. **Wire BFF** — `STUB_MODE=false`, kagent A2A endpoint, `proxy.url` routes agent LLM calls through agentgateway ✅
12. **Scripts** — setup, teardown, seed, demo

### Phase 4 — Ambient Mesh ✅
13. **Solo distribution of Istio** — installed in ambient mode (ztunnel DaemonSet at node level, no sidecars), `amss` namespace labeled `istio.io/dataplane-mode=ambient`, east-west mTLS confirmed ✅
14. **Demo tracks** — 4 tracks documented for different product combinations (agentgateway-only, +kagent, +ambient, full stack) ✅

Each step must have passing tests before moving to the next.

---

## Environment Variables

### Stores
| Var | Default | Description |
|---|---|---|
| `PORT` | 8081/8082/8083 | Listen port |
| `SEED_PATH` | `seed-data/<name>.json` | Path to seed JSON |

### MCP Servers
| Var | Default | Description |
|---|---|---|
| `KB_STORE_URL` | `http://kb-store:8081` | KB store base URL |
| `TICKET_STORE_URL` | `http://ticket-store:8082` | Ticket store base URL |

### BFF
| Var | Default | Description |
|---|---|---|
| `PORT` | 8080 | Listen port |
| `KB_STORE_URL` | `http://kb-store:8081` | KB store base URL |
| `TICKET_STORE_URL` | `http://ticket-store:8082` | Ticket store base URL |
| `CREW_STORE_URL` | `http://crew-store:8083` | Crew store base URL |
| `MISSION_SUPPORT_AGENT_URL` | `http://mission-support-agent:8090` | kagent agent endpoint (Phase 3) |
| `KB_CURATOR_AGENT_URL` | `http://kb-curator-agent:8091` | kagent agent endpoint (Phase 3) |
| `STUB_MODE` | `true` | When true (or agents unreachable), BFF returns keyword-matched stub responses |