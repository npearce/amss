# KB MCP — Knowledge Base Model Context Protocol Server

Go MCP server that wraps KB Store REST APIs as tools. Agents use these tools to search, read, create, and update KB articles.

## Build

```bash
cd mcp-servers/kb-mcp
go build -o kb-mcp .
```

## Test

```bash
go test ./... -v
```

## Run Locally

```bash
go run . [-port 9001] [-kb-store-url http://kb-store:8081]
```

Or with environment variables:
```bash
PORT=9001 KB_STORE_URL=http://kb-store:8081 go run .
```

## Run as Container

```bash
docker build -t kb-mcp:latest .
docker run -p 9001:9001 \
  -e KB_STORE_URL=http://kb-store:8081 \
  kb-mcp:latest
```

## MCP Tools

### `search_kb`
Search knowledge base articles.

**Arguments:**
- `query` (string) — search query
- `category` (string) — filter by category
- `tags` (string) — comma-separated tags
- `limit` (int) — max results
- `offset` (int) — pagination offset

**Returns:** List of matching articles with titles and IDs.

### `read_kb`
Get full details of an article by ID.

**Arguments:**
- `id` (string) — article ID (e.g., KB-001)

**Returns:** Full article with body, metadata, curator notes, etc.

### `create_kb`
Create a new KB article.

**Arguments:**
- `title` (string) — article title
- `body` (string) — article body (markdown)
- `category` (string) — category (required)
- `tags` (array) — tags
- `created_by` (string) — creator ID

**Returns:** Created article with auto-assigned ID.

### `update_kb`
Partially update a KB article.

**Arguments:**
- `id` (string) — article ID
- All other fields optional (title, body, category, tags, usefulness_score, duplicate_of, curator_notes, curator_tags)

**Returns:** Updated article.

## Architecture

- Connects to KB Store via HTTP (KB_STORE_URL env var)
- Exposes 4 tools following MCP protocol
- Tool results formatted as markdown for agent readability
- Error handling with descriptive messages

## Environment Variables

| Variable       | Default            | Description        |
|----------------|--------------------|-------------------|
| `PORT`         | `9001`             | Listen port        |
| `KB_STORE_URL` | `http://kb-store:8081` | KB Store base URL  |

## Dependencies

- Go 1.22+
- github.com/modelcontextprotocol/go-sdk (MCP SDK)
