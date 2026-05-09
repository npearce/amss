FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum . 2>/dev/null || true
RUN go mod download || true
COPY *.go .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bff .

FROM scratch

COPY --from=builder /app/bff /bff

EXPOSE 8080

ENV PORT=8080
ENV KB_STORE_URL=http://kb-store:8081
ENV TICKET_STORE_URL=http://ticket-store:8082
ENV CREW_STORE_URL=http://crew-store:8083
ENV MISSION_SUPPORT_AGENT_URL=http://mission-support-agent:8090
ENV KB_CURATOR_AGENT_URL=http://kb-curator-agent:8091

ENTRYPOINT ["/bff"]
