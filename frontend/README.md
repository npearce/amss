# AMSS Frontend

React + Vite single-page application for the Artemis Mission Support System.

Two views:
- **Astronaut Chat** (`/chat`) — AI-powered chat interface, sends queries to the Mission Support Agent via the BFF
- **Ground Control** (`/ground-control`) — Three-panel dashboard: ticket queue, KB articles, crew activity

Dark NASA-aesthetic design. No component library — hand-rolled CSS with design tokens.

## Prerequisites

- Node 20+
- BFF running at `localhost:8080` (or set `VITE_API_URL`)

## Development

```bash
npm install
npm run dev
```

Opens at `http://localhost:5173`. All BFF routes (`/articles`, `/tickets`, `/crew`, `/chat`, etc.) are proxied to `http://localhost:8080` automatically — no CORS issues.

## Production build

```bash
npm run build
npm run preview   # serve the built dist locally
```

## Environment

| Variable | Default | Description |
|---|---|---|
| `VITE_API_URL` | `""` (uses Vite proxy) | BFF base URL for production. Set to the BFF service URL, e.g. `http://bff.amss.svc.cluster.local:8080` |

In dev, leave `VITE_API_URL` unset — the Vite proxy handles routing to localhost:8080.

In k8s, set `VITE_API_URL` at Docker build time:
```bash
docker build \
  --build-arg VITE_API_URL=http://localhost:30080 \
  -t amss/frontend:latest .
```

## Container

```bash
# Build
docker build -t amss/frontend:latest .

# Run
docker run -p 3000:80 amss/frontend:latest
# then open http://localhost:3000
```

## Project structure

```
src/
├── main.jsx           # Router setup
├── App.jsx            # Shell: UserContext + NavBar + Outlet
├── UserContext.jsx    # Shared React context (currentUser, crew)
├── api.js             # BFF API client (all fetch calls)
├── views/
│   ├── AstronautChat.jsx    # Chat UI
│   └── GroundControl.jsx    # Three-panel dashboard
├── components/
│   ├── NavBar.jsx           # Top navigation bar
│   ├── UserSwitcher.jsx     # Crew member dropdown
│   ├── ChatMessage.jsx      # Chat bubble (user + assistant)
│   ├── TicketCard.jsx       # Ticket list item with expand
│   ├── ArticleCard.jsx      # KB article with expand + markdown
│   └── CrewRoster.jsx       # Crew list with activity expand
└── styles/
    └── global.css           # Dark theme + all component styles
```
