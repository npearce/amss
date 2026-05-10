# AMSS Specification — v0.1.0

## Overview

AMSS is a demo application showcasing Solo.io's commercial AI platform (Solo Enterprise for kagent, Solo Enterprise for agentgateway, Solo distribution of Istio) through a fictional NASA Artemis mission support system. This spec defines data schemas, API contracts, and store behavior.

See CLAUDE.md for ground rules (language, module structure, testing, build order).

---

## Data Stores — In-Memory Design

Each store (KB, Ticket, Crew) follows the same pattern:

1. On startup, read seed JSON from `SEED_PATH` into memory
2. Build a primary index: `map[string]*T` keyed by document ID for O(1) lookup
3. Build an inverted text index: lowercase tokenized words → `map[string]map[string]bool` (word → set of document IDs)
4. All reads and writes go through the in-memory structures
5. Write operations (create, update, delete) update both the primary map and the inverted index
6. `POST /reset` reloads seed JSON and rebuilds all indexes
7. No persistence to disk — ephemeral by design
8. Thread-safe: use `sync.RWMutex` — read lock for queries, write lock for mutations

### Text Search Behavior

- Tokenize query into lowercase words (split on whitespace and punctuation)
- For each token, look up matching document IDs in the inverted index
- Union (OR) all matching sets
- Rank results by number of matching tokens (more matches = higher rank)
- When filters are combined with search: apply filters first to get a candidate set, then intersect with search results

### Indexed Fields

- **KB Store**: title, body
- **Ticket Store**: title, description
- **Crew Store**: name, role, specialty (roster search); query, response (conversation search)

---

## API Envelope

Every response uses this envelope:

```json
{"data": <payload>, "error": null}
```

Error responses use HTTP status codes AND the envelope:

```json
{"data": null, "error": {"code": "NOT_FOUND", "message": "Article KB-999 not found"}}
```

Standard error codes:
- `NOT_FOUND` (404) — resource does not exist
- `BAD_REQUEST` (400) — invalid input, missing required fields
- `INTERNAL_ERROR` (500) — unexpected server error

---

## KB Store API

**Port**: 8081  
**Module**: `github.com/npearce/amss/stores/kb-store`

### Data Types

```go
type Article struct {
    ID                  string   `json:"id"`                     // "KB-001"
    Title               string   `json:"title"`
    Body                string   `json:"body"`                   // Markdown content
    Category            string   `json:"category"`               // See categories below
    Tags                []string `json:"tags"`                   // Author-supplied tags
    CreatedBy           string   `json:"created_by"`             // crew_id or "ground-control"
    CreatedAt           string   `json:"created_at"`             // RFC3339
    UpdatedAt           string   `json:"updated_at"`             // RFC3339
    ReferencedByTickets []string `json:"referenced_by_tickets"`  // Ticket IDs
    ReferenceCount      int      `json:"reference_count"`        // Denormalized count
    UsefulnessScore     *float64 `json:"usefulness_score"`       // 0.0-1.0, set by curator
    DuplicateOf         *string  `json:"duplicate_of"`           // KB ID if flagged as dupe
    CuratorNotes        *string  `json:"curator_notes"`
    CuratorTags         []string `json:"curator_tags"`
    LastCuratedAt       *string  `json:"last_curated_at"`        // RFC3339
}
```

### Categories

`life-support`, `navigation`, `comms`, `power`, `propulsion`, `eva`, `medical`, `operations`, `thermal`, `structures`

### Endpoints

#### `GET /health`
Returns: `{"status": "ok", "store": "kb-store"}`

#### `GET /articles`
List and search articles.

Query parameters:
| Param | Type | Description |
|---|---|---|
| `category` | string | Filter by category |
| `tags` | string | Comma-separated, match any (OR) across `tags` + `curator_tags` |
| `search` | string | Full-text search across title and body |
| `limit` | int | Max results (default 50, max 200) |
| `offset` | int | Pagination offset (default 0) |

Filters apply first, then search narrows within the filtered set. Results sorted by search relevance when `search` is present, otherwise by ID.

Response:
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

#### `GET /articles/{id}`
Get a single article by ID.

Response: `{"data": <Article>, "error": null}`  
404: `{"data": null, "error": {"code": "NOT_FOUND", "message": "Article KB-999 not found"}}`

#### `POST /articles`
Create a new article. ID is auto-assigned (next sequential KB-NNN).

Request body:
```json
{
  "title": "required",
  "body": "required",
  "category": "required — must be a valid category",
  "tags": ["optional"],
  "created_by": "optional, defaults to ground-control"
}
```

Response (201): `{"data": <Article>, "error": null}`  
400 if missing required fields or invalid category.

#### `PUT /articles/{id}`
Partial update. Only provided fields are changed.

Request body (all fields optional):
```json
{
  "title": "...",
  "body": "...",
  "category": "...",
  "tags": ["..."],
  "usefulness_score": 0.85,
  "duplicate_of": "KB-001",
  "curator_notes": "...",
  "curator_tags": ["..."]
}
```

- `updated_at` is always set on any update
- `last_curated_at` is set when any curator field is updated (curator_notes, curator_tags, usefulness_score, duplicate_of)

Response: `{"data": <Article>, "error": null}`  
404 if article not found.

#### `DELETE /articles/{id}`
Delete an article.

Response: `{"data": {"deleted": "KB-031"}, "error": null}`  
404 if not found.

#### `POST /reset`
Reload seed data and rebuild indexes.

Response: `{"data": {"message": "Reset to seed data", "article_count": 30}, "error": null}`

---

## Ticket Store API

**Port**: 8082  
**Module**: `github.com/npearce/amss/stores/ticket-store`

### Data Types

```go
type Ticket struct {
    ID                   string    `json:"id"`                      // "AMSS-001"
    Title                string    `json:"title"`
    Description          string    `json:"description"`
    Severity             string    `json:"severity"`                // P1, P2, P3, P4
    Status               string    `json:"status"`                  // open, in-progress, resolved, closed
    Category             string    `json:"category"`                // Same categories as KB
    ReportedBy           string    `json:"reported_by"`             // crew_id
    AssignedTo           string    `json:"assigned_to"`
    Mission              string    `json:"mission"`
    CreatedAt            string    `json:"created_at"`              // RFC3339
    UpdatedAt            string    `json:"updated_at"`              // RFC3339
    ResolvedAt           *string   `json:"resolved_at"`             // RFC3339, set on resolve
    KBArticlesReferenced []string  `json:"kb_articles_referenced"`
    Resolution           *string   `json:"resolution"`
    Comments             []Comment `json:"comments"`
}

type Comment struct {
    Author    string `json:"author"`
    Timestamp string `json:"timestamp"` // RFC3339
    Text      string `json:"text"`
}
```

### Severity Definitions

- **P1 — Critical/Safety**: Immediate crew safety risk. Cabin depressurization, O2 failure, abort.
- **P2 — Major/Degraded**: System degraded, crew safe, time-limited workaround. WCS failure, comms loss.
- **P3 — Minor**: Non-critical, no immediate impact. Sensor drift, display fault.
- **P4 — Informational**: Questions, requests, scheduling.

### Endpoints

#### `GET /health`
Returns: `{"status": "ok", "store": "ticket-store"}`

#### `GET /tickets`
List and filter tickets.

Query parameters:
| Param | Type | Description |
|---|---|---|
| `mission` | string | Filter by mission |
| `severity` | string | Filter by severity (P1-P4) |
| `status` | string | Filter by status |
| `category` | string | Filter by category |
| `search` | string | Full-text search across title and description |
| `limit` | int | Max results (default 50, max 200) |
| `offset` | int | Pagination offset (default 0) |

Response:
```json
{
  "data": {
    "tickets": [...],
    "total": 15,
    "limit": 50,
    "offset": 0
  },
  "error": null
}
```

#### `GET /tickets/{id}`
Get a single ticket with all comments.

#### `POST /tickets`
Create a new ticket. ID is auto-assigned (AMSS-NNN).

Request body:
```json
{
  "title": "required",
  "description": "required",
  "severity": "required — P1, P2, P3, or P4",
  "category": "required",
  "reported_by": "required",
  "mission": "required",
  "assigned_to": "optional, defaults to ground-control",
  "kb_articles_referenced": ["optional"]
}
```

Response (201). Status defaults to `open`.

#### `PUT /tickets/{id}`
Partial update.

Request body (all fields optional):
```json
{
  "status": "resolved",
  "severity": "P2",
  "assigned_to": "gc-eclss",
  "resolution": "Fixed by...",
  "kb_articles_referenced": ["KB-001"]
}
```

- `updated_at` is always set
- When `status` changes to `resolved` and `resolved_at` is null, `resolved_at` is set automatically

#### `POST /tickets/{id}/comments`
Add a comment to a ticket.

Request body:
```json
{
  "author": "required",
  "text": "required"
}
```

Response (201): `{"data": <Comment>, "error": null}`  
`updated_at` on the ticket is also set.

#### `POST /reset`
Reload seed data.

---

## Crew Store API

**Port**: 8083  
**Module**: `github.com/npearce/amss/stores/crew-store`

### Data Types

```go
type CrewMember struct {
    ID        string `json:"id"`        // "wiseman-r"
    Name      string `json:"name"`
    Role      string `json:"role"`      // Commander, Pilot, Mission Specialist, etc.
    Mission   string `json:"mission"`   // "artemis-ii", "all" for ground control
    Specialty string `json:"specialty"`
    Status    string `json:"status"`    // active, standby, rotated-out
    Persona   string `json:"persona"`   // astronaut, ground-control
}

type Conversation struct {
    ID                   string   `json:"id"`                      // "conv-0001"
    CrewID               string   `json:"crew_id"`
    Mission              string   `json:"mission"`
    SessionID            string   `json:"session_id"`
    Timestamp            string   `json:"timestamp"`               // RFC3339
    Query                string   `json:"query"`
    Response             string   `json:"response"`
    KBArticlesReferenced []string `json:"kb_articles_referenced"`
    TicketCreated        *string  `json:"ticket_created"`
}
```

### Endpoints

#### `GET /health`
Returns: `{"status": "ok", "store": "crew-store"}`

#### `GET /crew`
List crew members.

Query parameters:
| Param | Type | Description |
|---|---|---|
| `mission` | string | Filter by mission. Ground control (mission="all") is included in every mission filter. |
| `persona` | string | Filter by persona (astronaut, ground-control) |
| `status` | string | Filter by status |

Response:
```json
{
  "data": {
    "crew": [...],
    "total": 20
  },
  "error": null
}
```

#### `GET /crew/{id}`
Get a single crew member.

#### `GET /crew/{id}/activity`
Get recent conversations for a crew member.

Query parameters:
| Param | Type | Description |
|---|---|---|
| `limit` | int | Max results (default 20, max 100) |

Response:
```json
{
  "data": {
    "crew_id": "wiseman-r",
    "conversations": [...],
    "total": 5
  },
  "error": null
}
```

Conversations are sorted newest-first.

#### `GET /conversations`
List conversations with filters.

Query parameters:
| Param | Type | Description |
|---|---|---|
| `crew_id` | string | Filter by crew member |
| `mission` | string | Filter by mission |
| `session_id` | string | Filter by session |
| `limit` | int | Max results (default 50, max 200) |
| `offset` | int | Pagination offset (default 0) |

#### `POST /conversations`
Log a new conversation.

Request body:
```json
{
  "crew_id": "required — must exist in roster",
  "mission": "required",
  "session_id": "required",
  "query": "required",
  "response": "required",
  "kb_articles_referenced": ["optional"],
  "ticket_created": "optional"
}
```

Response (201). ID is auto-assigned (conv-NNNN). Timestamp is server-generated.  
404 if `crew_id` doesn't exist in the roster.

#### `POST /reset`
Reload seed data (roster + empty conversation list).

---

## Seed Data Files

Three JSON files in each store's `seed-data/` directory. These are the complete data sets loaded on startup and on `/reset`.

### crew.json

20 crew members. No conversations (empty array — conversations are created at runtime).

### kb.json

30 articles across 10 categories. Key features:
- KB-001, KB-018, KB-026, KB-030 all cover WCS/toilet topics (intentional near-duplicates for curator demo)
- Some articles have empty `tags` arrays (curator should populate them)
- Some articles have zero `reference_count` (curator should score them low)
- All `usefulness_score`, `duplicate_of`, `curator_notes`, `curator_tags`, `last_curated_at` fields start as null/empty

### tickets.json

15 tickets across missions and severities:
- 2× P1 (Ka-band comms dropout, solar array gimbal)
- 4× P2 (WCS toilet pressure, thermal radiator, O2 shutdown, EVA suit comms)
- 7× P3 (CO2 scrubber, star tracker, WCS bags, stowage, water quality, ARED exercise, nav display)
- 2× P4 (meal rotation, PAO photos)

AMSS-001 (WCS toilet) has the richest comment thread (5 comments, full resolution narrative).