#!/usr/bin/env bash
# demo-track2.sh — agentgateway + kagent agents demo track.
# Run from the repo root: ./scripts/demo-track2.sh
set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=============================================="
echo "  AMSS Demo — Track 2: agentgateway + kagent"
echo "  End-to-end AI agents with full observability"
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
echo "  1. Open the AMSS app — ask 'What is the WCS flush procedure?'"
echo "     (real Claude Sonnet 4.6 response, cites KB-001)"
echo "  2. Open Solo Enterprise UI at http://localhost:4000"
echo "     NOTE: This single UI shows BOTH agentgateway views AND kagent views."
echo "     Use the left nav to switch between the two product areas."
echo "     - agentgateway section: inbound routes, LLM egress traffic, token counts"
echo "     - kagent/Agents section: agent status, traces, MCP tool discovery"
echo "     kubectl get agents -n kagent"
echo "     kubectl get remotemcpserver -n amss"
echo "     kubectl get modelconfig -n kagent -o yaml"
echo "  3. Show traces in the kagent section: LLM call → tool invocation → KB search → response"
echo "  4. Show: activity generator creating tickets and KB articles in real time"
echo ""
echo "  DON'T SHOW: mesh details, ztunnel, mTLS, istio-system"
echo ""
echo "  TALKING POINTS:"
echo "  - 'Agent is a declarative YAML CRD — no Python, no server to manage'"
echo "  - 'MCP tools are discovered automatically via RemoteMCPServer CRD'"
echo "  - 'Every tool call, every token — visible in the trace'"
echo "  - 'LLM calls route through agentgateway — guardrails and failover apply'"
echo "  - 'Switch the model by changing one field in ModelConfig'"
echo ""
echo "  KUBECTL CHEAT SHEET:"
echo "    kubectl get agents -n kagent"
echo "    kubectl get remotemcpserver -n amss"
echo "    kubectl get modelconfig -n kagent -o yaml"
echo "    kubectl logs -l app=kagent-controller -n kagent --tail=20"
echo ""
echo "  DIRECT A2A CURL (bypass BFF — shows raw agent protocol):"
echo "    kubectl port-forward svc/kagent-controller -n kagent 8083:8083 &"
echo "    curl --max-time 120 -X POST http://localhost:8083/api/a2a/kagent/mission-support-agent/ \\"
echo "      -H 'Content-Type: application/json' \\"
echo "      -d '{\"jsonrpc\":\"2.0\",\"method\":\"message/send\",\"id\":\"demo-1\",\"params\":{\"message\":{\"role\":\"user\",\"parts\":[{\"kind\":\"text\",\"text\":\"What is the WCS flush procedure?\"}]}}}'"
echo ""
echo "  Press Ctrl+C to stop the demo"
echo ""

# Wait for Ctrl+C
trap "echo ''; echo 'Stopping demo...'; kill $AG_PID 2>/dev/null; pkill -f 'port-forward' 2>/dev/null; echo 'Done.'" EXIT
wait
