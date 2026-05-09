FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kb-mcp .

FROM scratch

COPY --from=builder /app/kb-mcp /kb-mcp

EXPOSE 9001

ENV PORT=9001
ENV KB_STORE_URL=http://kb-store:8081

ENTRYPOINT ["/kb-mcp"]
