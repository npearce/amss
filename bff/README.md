# BFF — Backend for Frontend

The BFF is the single entry point for the AMSS frontend and activity generator. It proxies store CRUD operations, routes chat requests to the Mission Support Agent, and routes curation requests to the KB Curator Agent.

**Port**: 8080

## API

### Health
- `GET /health` — `{"status":"ok","service":"bff"}`

### KB Articles (proxied to kb-store)
- `GET /articles[?category=&tags=&search=&limit=&offset=]`
- `GET /articles/{id}`
- `POST /articles`
- `PUT /articles/{id}`
- `DELETE /articles/{id}`

### Tickets (proxied to ticket-store)
- `GET /tickets[?mission=&severity=&status=&category=&search=&limit=&offset=]`
- `GET /tickets/{id}`
- `POST /tickets`
- `PUT /tickets/{id}`
- `POST /tickets/{id}/comments`

### Crew (proxied to crew-store)
- `GET /crew[?mission=&persona=&status=]`
- `GET /crew/{id}`
- `GET /crew/{id}/activity`
- `GET /conversations[?crew_id=&mission=&session_id=&limit=&offset=]`
- `POST /conversations`

### Chat
- `POST /chat` — sends a message to the Mission Support Agent and logs the conversation

Request:
```json
{
  "crew_id": "wiseman-r",
  "session_id": "sess-001",
  "mission": "artemis-ii",
  "message": "What is the WCS pressure status?"
}
```

Response:
```json
{
  "data": {
    "response": "...",
    "kb_articles_referenced": ["KB-001"],
    "ticket_created": null
  },
  "error": null
}
```

### Curate
- `POST /curate` — triggers the KB Curator Agent (body forwarded as-is)

### Reset
- `POST /reset` — resets all three stores to seed data

## Build

```bash
go build -o bff .
```

## Test

```bash
go test ./... -v
```

## Run locally

```bash
KB_STORE_URL=http://localhost:8081 \
TICKET_STORE_URL=http://localhost:8082 \
CREW_STORE_URL=http://localhost:8083 \
MISSION_SUPPORT_AGENT_URL=http://localhost:8090 \
KB_CURATOR_AGENT_URL=http://localhost:8091 \
go run .
```

## Run as container

```bash
docker build -t amss-bff .

docker run -p 8080:8080 \
  -e KB_STORE_URL=http://host.docker.internal:8081 \
  -e TICKET_STORE_URL=http://host.docker.internal:8082 \
  -e CREW_STORE_URL=http://host.docker.internal:8083 \
  -e MISSION_SUPPORT_AGENT_URL=http://host.docker.internal:8090 \
  -e KB_CURATOR_AGENT_URL=http://host.docker.internal:8091 \
  amss-bff
```

## Deploy to k8s

See `helm/amss/values.yaml` for service configuration. The BFF is exposed via agentgateway as the ingress entry point.
