#!/usr/bin/env bash
# demo-track1.sh — agentgateway only demo track.
# Run from the repo root: ./scripts/demo-track1.sh
set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=============================================="
echo "  AMSS Demo — Track 1: agentgateway"
echo "  Govern and observe AI traffic"
echo "=============================================="
echo ""

# Kill any existing port-forwards
pkill -f "port-forward" 2>/dev/null || true
sleep 1

# Start port-forwards
echo "==> Starting port-forwards..."
kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &
kubectl port-forward svc/solo-enterprise-ui -n kagent 4000:80 &
sleep 3

# Start activity generator in background
echo "==> Starting activity generator (22-min cycles)..."
cd activity-generator && BFF_URL=http://localhost:8080 go run . &
AG_PID=$!
cd ..

echo ""
echo "=============================================="
echo "  Demo is running!"
echo "=============================================="
echo ""
echo "  AMSS Application:     http://localhost:8080"
echo "  Solo Enterprise UI:   http://localhost:4000"
echo ""
echo "  WHAT TO SHOW:"
echo "  1. Open the AMSS app — Astronaut Chat, ask about WCS flush procedure"
echo "  2. Open Solo Enterprise UI — agentgateway section"
echo "  3. Show: ingress routes (frontend + BFF API)"
echo "  4. Show: LLM egress traffic (Anthropic calls, token counts, latency)"
echo "  5. Show: activity generator traffic flowing through gateway"
echo ""
echo "  DON'T SHOW: kagent UI, agent tracing, mesh details"
echo ""
echo "  TALKING POINTS:"
echo "  - 'Every request from the browser flows through agentgateway'"
echo "  - 'Every LLM call from the agent routes out through agentgateway'"
echo "  - 'One control plane for both inbound API traffic and outbound AI traffic'"
echo "  - 'Token counts, latency, model routing — all observable here'"
echo ""
echo "  QUICK CURL DEMO:"
echo "    curl -s http://localhost:8080/api/v1/chat \\"
echo "      -X POST -H 'Content-Type: application/json' \\"
echo "      -d '{\"crew_id\":\"wiseman-r\",\"session_id\":\"demo-1\",\"mission\":\"artemis-ii\",\"message\":\"WCS pressure is dropping\"}' \\"
echo "      | jq '.data'"
echo ""
echo "  Press Ctrl+C to stop the demo"
echo ""

# Wait for Ctrl+C
trap "echo ''; echo 'Stopping demo...'; kill $AG_PID 2>/dev/null; pkill -f 'port-forward' 2>/dev/null; echo 'Done.'" EXIT
wait
