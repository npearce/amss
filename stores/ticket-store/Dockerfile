FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY *.go ./
COPY seed-data/ seed-data/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ticket-store .

FROM scratch

COPY --from=builder /app/ticket-store /ticket-store
COPY --from=builder /app/seed-data/ /seed-data/

EXPOSE 8082

ENV PORT=8082
ENV SEED_PATH=seed-data/tickets.json

ENTRYPOINT ["/ticket-store"]
