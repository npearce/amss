# KB Store — Knowledge Base HTTP Server

In-memory HTTP server providing REST access to a knowledge base of 30 Artemis mission support articles. Implements full-text search with inverted indexing and curator tools for deduplication and scoring.

## Build

```bash
cd stores/kb-store
go build -o kb-store .
```

## Test

```bash
go test ./... -v
```

## Run Locally

```bash
go run . [-port 8081] [-seed seed-data/kb.json]
```

Or with environment variables:
```bash
PORT=8081 SEED_PATH=seed-data/kb.json go run .
```

## Run as Container

Build the image:
```bash
docker build -t kb-store:latest .
```

Run:
```bash
docker run -p 8081:8081 kb-store:latest
```

With environment variables:
```bash
docker run -p 8081:8081 \
  -e PORT=8081 \
  -e SEED_PATH=/seed-data/kb.json \
  kb-store:latest
```

## API Endpoints

### `GET /health`
Health check.

**Response:**
```json
{"data": {"status": "ok", "store": "kb-store"}, "error": null}
```

### `GET /articles`
List articles with optional filtering and search.

**Query Parameters:**
- `category` (string) — filter by category
- `tags` (string) — comma-separated tags, match any (OR)
- `search` (string) — full-text search across title and body
- `limit` (int) — max results (default 50, max 200)
- `offset` (int) — pagination offset (default 0)

**Response:**
```json
{
  "data": {
    "articles": [...],
    "total": 30,
    "limit": 50,
    "offset": 0
  },
  "error": null
}
```

### `GET /articles/{id}`
Get a single article by ID.

**Response:**
```json
{"data": <Article>, "error": null}
```

### `POST /articles`
Create a new article. ID is auto-assigned (KB-NNN).

**Request:**
```json
{
  "title": "required",
  "body": "required",
  "category": "required",
  "tags": ["optional"],
  "created_by": "optional, defaults to ground-control"
}
```

**Response (201):**
```json
{"data": <Article>, "error": null}
```

### `PUT /articles/{id}`
Partial update. Only provided fields change.

**Request:**
```json
{
  "title": "...",
  "body": "...",
  "category": "...",
  "tags": [...],
  "usefulness_score": 0.85,
  "duplicate_of": "KB-001",
  "curator_notes": "...",
  "curator_tags": [...]
}
```

When any curator field (curator_notes, curator_tags, usefulness_score, duplicate_of) is updated, `last_curated_at` is set. `updated_at` is always set.

**Response:**
```json
{"data": <Article>, "error": null}
```

### `DELETE /articles/{id}`
Delete an article.

**Response:**
```json
{"data": {"deleted": "KB-001"}, "error": null}
```

### `POST /reset`
Reload seed data and rebuild indexes.

**Response:**
```json
{"data": {"message": "Reset to seed data", "article_count": 30}, "error": null}
```

## Architecture

### In-Memory Store

- Articles are stored in a map indexed by ID for O(1) lookup.
- An inverted index maps lowercase tokenized words (from title+body) to sets of article IDs.
- Thread-safe with `sync.RWMutex`: read lock for queries, write lock for mutations.

### Search

- Query is tokenized into lowercase words (split on whitespace and punctuation).
- For each token, matching document IDs are looked up in the inverted index.
- Results are unioned (OR across tokens) and ranked by match count.
- When filters are combined with search, filters apply first to narrow candidates, then search intersects with filtered set.

### Seed Data

On startup and on `/reset`, the store reads `SEED_PATH` (default `seed-data/kb.json`). Seed data contains 30 articles across 10 categories with intentional near-duplicates for curator demo.

### Categories

`life-support`, `navigation`, `comms`, `power`, `propulsion`, `eva`, `medical`, `operations`, `thermal`, `structures`

## Deployment

### Kubernetes

Helm chart coming soon. For now, apply raw manifests:

```bash
kubectl apply -f manifests/kb-store.yaml
```

## Environment Variables

| Variable  | Default              | Description        |
|-----------|----------------------|--------------------|
| `PORT`    | `8081`               | Listen port        |
| `SEED_PATH` | `seed-data/kb.json` | Path to seed JSON  |
