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

Solo Enterprise for kagent CRDs in k8s. Agents are `Agent` custom resources with `type: Declarative`, system prompts, and tool references to MCP servers. For local dev (docker-compose), agents run as plain Go HTTP servers that accept requests and call MCP tools. kagent supports Go ADK natively, so no Python dependency. Two agents:
- **Mission Support Agent** — answers crew questions, searches KB, creates tickets
- **KB Curator Agent** — de-duplicates articles, auto-tags, scores usefulness

### MCP Servers

Go, using the official SDK (`github.com/modelcontextprotocol/go-sdk/mcp`). Wrap the store REST APIs as MCP tools. Two servers:
- **KB MCP** — search, read, create, update KB articles
- **Ticket MCP** — search, read, create, update tickets, add comments

### Containers and Deployment

Each service has a Dockerfile with multi-stage build (build stage + scratch/distroless runtime). Each service README documents:
- How to build: `go build -o <name> .`
- How to test: `go test ./... -v`
- How to run locally: `go run . [env vars]`
- How to run as container: `docker build` and `docker run` with env vars
- How to deploy to k8s: pointer to helm values or raw manifest
- Local k8s: OrbStack (preferred) or kind

`docker-compose.yml` at the root runs the full stack locally (no Solo products required).

### k8s Deployment (Solo Enterprise Products)

For k8s deployment with the full Solo stack:
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
  Solo Enterprise for agentgateway (ingress)
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
   LLM providers             LLM providers
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
├── docker-compose.yml               # Full local stack
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
│   ├── kb-mcp/
│   │   ├── go.mod
│   │   ├── main.go
│   │   ├── tools.go                 # MCP tool definitions
│   │   ├── tools_test.go
│   │   ├── Dockerfile
│   │   └── README.md
│   │
│   └── ticket-mcp/                  # Same shape
│       ├── go.mod
│       ├── main.go
│       ├── tools.go
│       ├── tools_test.go
│       ├── Dockerfile
│       └── README.md
│
├── agents/
│   ├── mission-support-agent/
│   │   ├── go.mod
│   │   ├── main.go
│   │   ├── agent.go                 # Chat handler, MCP client calls
│   │   ├── agent_test.go
│   │   ├── prompts/
│   │   │   └── system.md
│   │   ├── Dockerfile
│   │   └── README.md
│   │
│   └── kb-curator-agent/            # Same shape
│       ├── go.mod
│       ├── main.go
│       ├── agent.go
│       ├── agent_test.go
│       ├── prompts/
│       │   └── system.md
│       ├── Dockerfile
│       └── README.md
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

## Build Order

### v0.1.0-alpha1 — Foundation (stores)
1. **Stores** (kb-store, ticket-store, crew-store) — no dependencies, test independently

### v0.1.0-alpha2 — Application layer
2. **MCP servers** (kb-mcp, ticket-mcp) — depend on stores at runtime
3. **BFF** — depends on stores and agents at runtime
4. **Agents** (mission-support, kb-curator) — depend on MCP servers at runtime
5. **Frontend** — depends on BFF at runtime
6. **Activity generator** — depends on BFF at runtime
7. **docker-compose.yml** — full local stack, no Solo products required

### v0.1.0-alpha3 — Solo Enterprise integration
8. **Helm charts** — k8s manifests, Solo Enterprise for kagent Agent CRDs, agentgateway config
9. **Solo Enterprise for kagent** install + agent deployment
10. **Solo Enterprise for agentgateway** install + LLM routing, guardrails, failover
11. **Scripts** — setup, teardown, seed, demo

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
| `MISSION_SUPPORT_AGENT_URL` | `http://mission-support-agent:8090` | Agent URL |
| `KB_CURATOR_AGENT_URL` | `http://kb-curator-agent:8091` | Agent URL |

### Agents
| Var | Default | Description |
|---|---|---|
| `PORT` | 8090/8091 | Listen port |
| `KB_MCP_URL` | `http://kb-mcp:9001` | KB MCP server URL |
| `TICKET_MCP_URL` | `http://ticket-mcp:9002` | Ticket MCP server URL |