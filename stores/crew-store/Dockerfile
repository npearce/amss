FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum . 2>/dev/null || true
RUN go mod download || true
COPY *.go .
COPY seed-data/ seed-data/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o crew-store .

FROM scratch

COPY --from=builder /app/crew-store /crew-store
COPY --from=builder /app/seed-data /seed-data

EXPOSE 8083

ENV PORT=8083
ENV SEED_PATH=seed-data/crew.json

ENTRYPOINT ["/crew-store"]
