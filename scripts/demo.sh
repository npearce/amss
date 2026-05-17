#!/usr/bin/env bash
# demo.sh — Start the AMSS demo (port-forwards + activity generator).
# Run from the repo root: ./scripts/demo.sh
set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Kill any existing port-forwards and activity generator
pkill -f "port-forward" 2>/dev/null || true
pkill -f "activity-generator" 2>/dev/null || true
sleep 1

# Start port-forwards
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
kubectl port-forward svc/solo-enterprise-ui -n kagent 4000:80 &
sleep 3

# Start activity generator in background
cd activity-generator && BFF_URL=http://localhost:8080 go run . &
AG_PID=$!
cd ..

echo ""
echo "  AMSS Application:   http://localhost:8080"
echo "  Solo Enterprise UI: http://localhost:4000"
echo ""
echo "  Activity generator running (22-min cycles). Press Ctrl+C to stop."
echo ""

trap "echo ''; kill $AG_PID 2>/dev/null; pkill -f 'port-forward' 2>/dev/null" EXIT
wait
