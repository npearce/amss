# Ticket MCP Server

MCP server that wraps the ticket store REST API as tools for use by AI agents. Built with the official Go MCP SDK and the kmcp scaffold pattern.

**Default transport**: stdio (for use with kagent / Claude Desktop)  
**Optional transport**: streamable HTTP via `--http` flag

## Tools

| Tool | Description |
|---|---|
| `search_tickets` | Search tickets by mission, severity, status, category, or text |
| `read_ticket` | Read a full ticket by ID, including comments |
| `create_ticket` | Create a new ticket (title, description, severity, category, reported_by, mission required) |
| `update_ticket` | Partial update — status, severity, assignee, resolution, KB refs, and more |
| `add_ticket_comment` | Add a comment to an existing ticket |
| `get_ticket_summary` | Aggregate count of all tickets by severity and status |

## Environment Variables

| Var | Default | Description |
|---|---|---|
| `TICKET_STORE_URL` | `http://ticket-store:8082` | Ticket store base URL |

## Build

```bash
go build -o server ./cmd/server
```

## Test

```bash
go test ./... -v
go test ./... -v -race
```

## Run locally (stdio mode)

```bash
TICKET_STORE_URL=http://localhost:8082 go run ./cmd/server
```

## Run locally (HTTP mode)

```bash
TICKET_STORE_URL=http://localhost:8082 go run ./cmd/server --http :9002
```

## Run as container

```bash
docker build -t amss-ticket-mcp .

# stdio mode (default)
docker run --rm -i amss-ticket-mcp

# HTTP mode
docker run -p 9002:9002 amss-ticket-mcp /app/server --http :9002
```

## Deploy to k8s

See `helm/amss/values.yaml` for service configuration. The ticket MCP server runs in-mesh with ambient mTLS alongside the ticket store.
