# Crew Store

In-memory crew roster and conversation log for AMSS. Loads 20 crew members from seed data on startup. The roster is read-only; only conversations are mutable. Exposes a REST API consumed by the BFF and agents.

**Port**: 8083

## Data

20 crew members across 4 missions:
- **Artemis II** (4 astronauts): Wiseman, Glover, Koch, Hansen
- **Artemis III** (4 astronauts): Patel, Okafor, Chen, Bergström
- **Artemis IV** (4 astronauts): Tanaka, Morales, Adeyemi, Kovács
- **Ground Control** (8): Torres, McKinnon, Vasquez, Ivanov, Webb, Nair, Osei, Volkov

Ground control members have `mission: "all"` and are included in every mission-specific filter.

Conversations are ephemeral — `POST /reset` restores the roster and clears all conversations.

## API

### Health
- `GET /health` — `{"status":"ok","store":"crew-store"}`

### Crew (read-only)
- `GET /crew[?mission=&search=&limit=&offset=]`
- `GET /crew/{id}`
- `GET /crew/{id}/activity[?limit=]`

### Conversations
- `GET /conversations[?crew_id=&mission=&session_id=&limit=&offset=]`
- `POST /conversations`
- `POST /reset`

### Crew filter parameters
| Param | Description |
|---|---|
| `mission` | Filter by mission (e.g., `artemis-ii`). Ground control (`mission=all`) always included. |
| `search` | Full-text search over name, role, specialty |
| `limit` | Max results (default 50, max 200) |
| `offset` | Pagination offset (default 0) |

### Activity parameters
| Param | Description |
|---|---|
| `limit` | Max conversations (default 20, max 100). Results sorted newest-first. |

### Conversation filter parameters
| Param | Description |
|---|---|
| `crew_id` | Filter by crew member ID |
| `mission` | Filter by mission |
| `session_id` | Filter by session ID |
| `limit` | Max results (default 50, max 200) |
| `offset` | Pagination offset (default 0) |

### Create conversation (POST /conversations)
Required: `crew_id`, `mission`, `session_id`, `query`, `response`  
Optional: `kb_articles_referenced`, `ticket_created`

Returns 400 if `crew_id` is not found in the roster. ID is auto-assigned (`conv-NNNN`). Timestamp is server-generated RFC3339 UTC.

## Build

```bash
go build -o crew-store .
```

## Test

```bash
go test ./... -v
go test ./... -v -race
```

## Run locally

```bash
PORT=8083 SEED_PATH=seed-data/crew.json go run .
```

## Run as container

```bash
docker build -t amss-crew-store .

docker run -p 8083:8083 amss-crew-store
```

## Deploy to k8s

See `helm/amss/values.yaml` for service configuration. The crew-store runs in-mesh with ambient mTLS.
