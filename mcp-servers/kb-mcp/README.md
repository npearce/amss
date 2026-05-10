# KB MCP Server

MCP server that wraps the KB store REST API as tools for use by AI agents. Built with the official Go MCP SDK and the kmcp scaffold pattern.

**Default transport**: stdio (for use with kagent / Claude Desktop)  
**Optional transport**: streamable HTTP via `--http` flag

## Tools

| Tool | Description |
|---|---|
| `search_kb` | Search articles by text, category, or tags |
| `read_kb_article` | Read full article by ID |
| `create_kb_article` | Create a new article (title, body, category required) |
| `update_kb_article` | Partial update — curator fields: usefulness_score, duplicate_of, curator_notes, curator_tags |
| `list_kb_categories` | Returns the 10 valid category values |

## Environment Variables

| Var | Default | Description |
|---|---|---|
| `KB_STORE_URL` | `http://kb-store:8081` | KB store base URL |

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
KB_STORE_URL=http://localhost:8081 go run ./cmd/server
```

## Run locally (HTTP mode)

```bash
KB_STORE_URL=http://localhost:8081 go run ./cmd/server --http :9001
```

## Run as container

```bash
docker build -t amss-kb-mcp .

# stdio mode (default)
docker run --rm -i amss-kb-mcp

# HTTP mode
docker run -p 9001:9001 amss-kb-mcp /app/server --http :9001
```

## Deploy to k8s

See `helm/amss/values.yaml` for service configuration. The KB MCP server runs in-mesh with ambient mTLS alongside the KB store.
