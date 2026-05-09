FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.mod
COPY *.go .
COPY seed-data/ seed-data/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kb-store .

FROM scratch

COPY --from=builder /app/kb-store /kb-store
COPY --from=builder /app/seed-data /seed-data

EXPOSE 8081

ENV PORT=8081
ENV SEED_PATH=/seed-data/kb.json

ENTRYPOINT ["/kb-store"]
