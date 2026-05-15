#!/usr/bin/env bash
# demo-track4.sh — Full stack demo (all products).
# Run from the repo root: ./scripts/demo-track4.sh
set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=============================================="
echo "  AMSS Demo — Track 4: Full Stack"
echo "  agentgateway + kagent + ambient mesh"
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
echo "  DEMO FLOW:"
echo ""
echo "  1. ASTRONAUT CHAT (show the app working)"
echo "     - Open http://localhost:8080"
echo "     - Select Reid Wiseman (Artemis II commander)"
echo "     - Send: 'WCS pressure is dropping, what do I do?'"
echo "     - Show the real Claude Sonnet 4.6 response citing KB-001"
echo ""
echo "  2. AGENTGATEWAY — ingress and LLM egress (Solo Enterprise UI at localhost:4000)"
echo "     - Open http://localhost:4000 → agentgateway section"
echo "     - NOTE: localhost:4000 shows BOTH agentgateway views AND kagent views"
echo "       (one UI, two product areas — use the left nav to switch between them)"
echo "     - Show: inbound routes (frontend /*, BFF /api/v1/*)"
echo "     - Show: LLM egress (Anthropic call — token count, latency, model)"
echo "     - Talking point: 'One gateway for both API traffic and AI traffic'"
echo ""
echo "  3. KAGENT — agents, tools, traces (Solo Enterprise UI at localhost:4000)"
echo "     - Same UI — http://localhost:4000 → switch to kagent/Agents section"
echo "     - Show agents READY:"
echo "         kubectl get agents -n kagent"
echo "     - Show MCP tool discovery:"
echo "         kubectl get remotemcpserver -n amss"
echo "     - Show trace: LLM call → search_kb → read_kb_article → response"
echo "     - Talking point: 'Agent is YAML, tools are MCP, observability is built in'"
echo ""
echo "  4. AMBIENT MESH — mTLS, no sidecars (Solo Enterprise UI + kubectl)"
echo "     - Show pods — all 1/1, zero sidecars:"
echo "         kubectl get pods -n amss"
echo "     - Show ztunnel DaemonSet:"
echo "         kubectl get pods -n istio-system"
echo "     - Generate traffic, show mTLS on the service graph:"
echo "         for i in \$(seq 1 5); do curl -s http://localhost:8080/api/v1/crew | jq '.data.total'; done"
echo "     - Talking point: 'Same mTLS you get in production, zero app changes'"
echo ""
echo "  5. KB CURATOR (show AI-driven operations)"
echo "     curl -s -X POST http://localhost:8080/api/v1/curator \\"
echo "       -H 'Content-Type: application/json' -d '{}' | jq '.data'"
echo "     - Show duplicates flagged (WCS articles KB-001, KB-018, KB-026, KB-030)"
echo ""
echo "  KUBECTL CHEAT SHEET:"
echo "    kubectl get agents -n kagent"
echo "    kubectl get remotemcpserver -n amss"
echo "    kubectl get modelconfig -n kagent -o yaml"
echo "    kubectl get pods -n amss"
echo "    kubectl get pods -n istio-system"
echo "    kubectl get namespace amss --show-labels"
echo "    kubectl get httproute -n amss -o wide"
echo "    kubectl get httproute -n agentgateway-system"
echo "    kubectl get agentgatewaybackend -n agentgateway-system"
echo ""
echo "  COMPLETE TRACE PATH (for the whiteboard):"
echo "    Browser → agentgateway (ingress)"
echo "      → BFF (mTLS via ztunnel)"
echo "        → kagent A2A"
echo "          → Claude Sonnet 4.6 via agentgateway (LLM egress)"
echo "            → KB MCP tools (mTLS via ztunnel)"
echo "              → KB Store (mTLS via ztunnel)"
echo "                → response"
echo ""
echo "  Press Ctrl+C to stop the demo"
echo ""

# Wait for Ctrl+C
trap "echo ''; echo 'Stopping demo...'; kill $AG_PID 2>/dev/null; pkill -f 'port-forward' 2>/dev/null; echo 'Done.'" EXIT
wait
