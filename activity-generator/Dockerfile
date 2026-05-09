# ── Build stage ──────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o activity-generator .

# ── Runtime stage ─────────────────────────────────────────────
FROM scratch

COPY --from=builder /app/activity-generator /activity-generator
COPY --from=builder /app/scenarios.json /scenarios.json

ENV BFF_URL=http://bff:8080
ENV SCENARIO_FILE=/scenarios.json
ENV PAUSE_SECONDS=5

ENTRYPOINT ["/activity-generator"]
