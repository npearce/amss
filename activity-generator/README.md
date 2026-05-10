# Activity Generator

Drives the AMSS BFF with scripted scenarios to simulate crew and ground control activity. Each scenario is a sequence of HTTP requests; `{{prev.<key>}}` placeholders in paths and bodies are resolved from the previous step's response.

## Prerequisites

- Go 1.22+
- BFF running (default: `http://localhost:8080`)

## Build

```bash
go build -o activity-generator .
```

## Test

```bash
go test ./... -v -race
```

## Run

```bash
# defaults — BFF at localhost:8080, 5s pause between scenarios
./activity-generator

# override
BFF_URL=http://localhost:8080 \
SCENARIO_FILE=scenarios.json \
PAUSE_SECONDS=2 \
./activity-generator
```

Runs each scenario once in order, then exits. Non-2xx responses abort the current scenario and continue to the next.

## Environment

| Var | Default | Description |
|---|---|---|
| `BFF_URL` | `http://localhost:8080` | BFF base URL |
| `SCENARIO_FILE` | `scenarios.json` | Path to scenarios JSON |
| `PAUSE_SECONDS` | `5` | Pause between scenarios (seconds) |

## Scenario format

```json
[
  {
    "name": "Human-readable name",
    "steps": [
      {
        "method": "POST",
        "path": "/tickets",
        "body": { "title": "...", "severity": "P3" },
        "description": "what this step does"
      },
      {
        "method": "PATCH",
        "path": "/tickets/{{prev.id}}",
        "body": { "status": "closed" },
        "description": "close using the id from step 1"
      }
    ]
  }
]
```

`{{prev.<key>}}` is replaced with the value of `<key>` from the `data` field of the previous step's response envelope.

## Container

```bash
# Build
docker build -t amss/activity-generator:latest .

# Run
docker run --rm \
  -e BFF_URL=http://host.docker.internal:8080 \
  amss/activity-generator:latest
```

## Kubernetes

The activity-generator runs as a Job (runs once to completion). See `../k8s/activity-generator.yaml`.

```bash
kubectl apply -f ../k8s/activity-generator.yaml -n amss
kubectl logs job/activity-generator -n amss
```
