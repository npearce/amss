FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.mod
COPY go.sum . 2>/dev/null || true
RUN go mod download || true
COPY *.go .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kb-mcp .

FROM scratch

COPY --from=builder /app/kb-mcp /kb-mcp

EXPOSE 9001

ENV PORT=9001
ENV KB_STORE_URL=http://kb-store:8081

ENTRYPOINT ["/kb-mcp"]
