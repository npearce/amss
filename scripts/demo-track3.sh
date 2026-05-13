#!/usr/bin/env bash
# demo-track3.sh — agentgateway + ambient mesh demo track.
# Run from the repo root: ./scripts/demo-track3.sh
set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=============================================="
echo "  AMSS Demo — Track 3: agentgateway + ambient mesh"
echo "  Zero-trust east-west mTLS, no sidecars"
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
echo "  1. Pods — all 1/1, no sidecars injected:"
echo "     kubectl get pods -n amss"
echo "  2. Istio running (ztunnel DaemonSet, no sidecars):"
echo "     kubectl get pods -n istio-system"
echo "  3. Namespace enrolled:"
echo "     kubectl get namespace amss --show-labels"
echo "  4. Open Solo Enterprise UI — mesh observability view"
echo "     Send traffic to light up the service graph:"
echo "       for i in \$(seq 1 5); do curl -s http://localhost:8080/api/v1/crew | jq '.data.total'; done"
echo "  5. Show mTLS badges on all amss east-west edges in the UI"
echo ""
echo "  DON'T SHOW: kagent UI, agent traces, agent CRDs, Python ADK"
echo ""
echo "  TALKING POINTS:"
echo "  - 'Every pod is still 1/1 — no sidecars, no restarts, zero app changes'"
echo "  - 'ztunnel runs as a DaemonSet at the node level — transparent to workloads'"
echo "  - 'BFF to kb-store, bff to ticket-store, mcp-servers to stores — all mTLS'"
echo "  - 'The application code has zero crypto, zero cert rotation logic'"
echo "  - 'This is the same mesh enforcement you get in production, on a laptop'"
echo ""
echo "  KUBECTL CHEAT SHEET:"
echo "    kubectl get pods -n amss"
echo "    kubectl get pods -n istio-system"
echo "    kubectl get namespace amss --show-labels"
echo "    kubectl get pods -n istio-system -l app=ztunnel -o wide"
echo ""
echo "  GENERATE TRAFFIC (to populate the mesh graph):"
echo "    for i in \$(seq 1 10); do"
echo "      curl -s http://localhost:8080/api/v1/crew | jq '.data.total'"
echo "      curl -s http://localhost:8080/api/v1/kb | jq '.data.total'"
echo "    done"
echo ""
echo "  Press Ctrl+C to stop the demo"
echo ""

# Wait for Ctrl+C
trap "echo ''; echo 'Stopping demo...'; kill $AG_PID 2>/dev/null; pkill -f 'port-forward' 2>/dev/null; echo 'Done.'" EXIT
wait
