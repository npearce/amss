# Ticket Store

In-memory ticket store for AMSS. Loads seed data on startup, supports full-text search on title and description, and exposes a REST API consumed by the Ticket MCP server.

**Port**: 8082

## Data

15 seed tickets across 4 severities (P1–P4) and 10 categories. AMSS-001 has a 5-comment resolution thread. All data is ephemeral — `POST /reset` restores the seed state.

## API

### Health
- `GET /health` — `{"status":"ok","store":"ticket-store"}`

### Tickets
- `GET /tickets[?mission=&severity=&status=&category=&search=&limit=&offset=]`
- `GET /tickets/{id}`
- `POST /tickets`
- `PUT /tickets/{id}`
- `POST /tickets/{id}/comments`
- `POST /reset`

### Filter parameters
| Param | Description |
|---|---|
| `mission` | Filter by mission (e.g., `artemis-ii`) |
| `severity` | P1, P2, P3, or P4 |
| `status` | open, in-progress, resolved, closed |
| `category` | One of 10 valid categories |
| `search` | Full-text search over title and description |
| `limit` | Max results (default 50, max 200) |
| `offset` | Pagination offset (default 0) |

Filters narrow first, then search applies within the filtered set.

### Create ticket (POST /tickets)
Required: `title`, `description`, `severity`, `category`, `reported_by`, `mission`  
Optional: `assigned_to` (defaults to `ground-control`), `kb_articles_referenced`

Status is always `open` on create. ID is auto-assigned (`AMSS-NNN`).

### Update ticket (PUT /tickets/{id})
Partial update. All fields optional: `status`, `severity`, `assigned_to`, `resolution`, `kb_articles_referenced`, `title`, `description`, `category`, `mission`.

When `status` changes to `resolved` and `resolved_at` is null, `resolved_at` is set automatically.

### Add comment (POST /tickets/{id}/comments)
Required: `author`, `text`. Timestamp is server-generated. Sets `updated_at` on the ticket.

## Build

```bash
go build -o ticket-store .
```

## Test

```bash
go test ./... -v
go test ./... -v -race
```

## Run locally

```bash
PORT=8082 SEED_PATH=seed-data/tickets.json go run .
```

## Run as container

```bash
docker build -t amss-ticket-store .

docker run -p 8082:8082 amss-ticket-store
```

## Deploy to k8s

See `helm/amss/values.yaml` for service configuration. The ticket-store runs in-mesh with ambient mTLS.
