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

# Read Keycloak credentials for the activity generator
KC_CLIENT=""
KC_SECRET=""
KC_URL=""
if kubectl get secret keycloak-client -n amss &>/dev/null; then
  KC_CLIENT=$(kubectl get secret keycloak-client -n amss -o jsonpath='{.data.client-id}' | base64 -d)
  KC_SECRET=$(kubectl get secret keycloak-client -n amss -o jsonpath='{.data.client-secret}' | base64 -d)
  KC_URL=$(kubectl get secret keycloak-client -n amss -o jsonpath='{.data.keycloak-url}' | base64 -d)
  echo "  Keycloak credentials loaded for activity generator."
fi

# Start activity generator in background
cd activity-generator && \
  BFF_URL=http://localhost:8080 \
  KEYCLOAK_URL="${KC_URL}" \
  KEYCLOAK_CLIENT_ID="${KC_CLIENT}" \
  KEYCLOAK_CLIENT_SECRET="${KC_SECRET}" \
  KEYCLOAK_USERNAME="amss-agent" \
  KEYCLOAK_PASSWORD="agent-secret" \
  go run . &
AG_PID=$!
cd ..

echo ""
echo "  AMSS Application:   http://localhost:8080"
echo "  Solo Enterprise UI: http://localhost:4000"
echo ""
echo "  Activity generator running (22-min cycles). Press Ctrl+C to stop."
echo ""

if [ -n "$KC_URL" ]; then
  echo "  Get a fresh token:"
  echo "    curl -s -d \"client_id=${KC_CLIENT}\" -d \"client_secret=${KC_SECRET}\" \\"
  echo "      -d \"username=wiseman\" -d \"password=artemis\" -d \"grant_type=password\" \\"
  echo "      \"${KC_URL}/realms/master/protocol/openid-connect/token\" | jq -r .access_token"
  echo ""
fi

trap "echo ''; kill $AG_PID 2>/dev/null; pkill -f 'port-forward' 2>/dev/null" EXIT
wait
