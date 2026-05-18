#!/usr/bin/env bash
# deploy.sh — Build all AMSS images and deploy to the local k8s cluster.
# Run from the repo root: ./k8s/deploy.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "==> Building container images..."
docker build -t amss/kb-store:latest     ./stores/kb-store
docker build -t amss/ticket-store:latest ./stores/ticket-store
docker build -t amss/crew-store:latest   ./stores/crew-store
docker build -t amss/kb-mcp:latest       ./mcp-servers/kb-mcp
docker build -t amss/ticket-mcp:latest   ./mcp-servers/ticket-mcp
docker build -t amss/bff:latest          ./bff
# Detect Keycloak credentials — baked into the frontend at build time by Vite.
# If the keycloak-client secret doesn't exist, vars are empty and auth is disabled.
KC_FE_URL=""
KC_FE_CLIENT=""
KC_FE_SECRET=""
if kubectl get secret keycloak-client -n amss &>/dev/null; then
  KC_FE_URL=$(kubectl get secret keycloak-client -n amss -o jsonpath='{.data.keycloak-url}' | base64 -d)
  KC_FE_CLIENT=$(kubectl get secret keycloak-client -n amss -o jsonpath='{.data.client-id}' | base64 -d)
  KC_FE_SECRET=$(kubectl get secret keycloak-client -n amss -o jsonpath='{.data.client-secret}' | base64 -d)
fi
docker build -t amss/frontend:latest \
  --build-arg VITE_KEYCLOAK_URL="${KC_FE_URL}" \
  --build-arg VITE_KEYCLOAK_CLIENT_ID="${KC_FE_CLIENT}" \
  --build-arg VITE_KEYCLOAK_CLIENT_SECRET="${KC_FE_SECRET}" \
  ./frontend

# --- kind users: uncomment to load images into the cluster ---
# kind load docker-image amss/kb-store:latest
# kind load docker-image amss/ticket-store:latest
# kind load docker-image amss/crew-store:latest
# kind load docker-image amss/kb-mcp:latest
# kind load docker-image amss/ticket-mcp:latest
# kind load docker-image amss/bff:latest
# kind load docker-image amss/frontend:latest

echo ""
echo "==> Applying manifests..."
kubectl apply -f k8s/namespace.yaml

# Stores first — MCP servers and BFF depend on them at runtime
kubectl apply -f k8s/kb-store.yaml
kubectl apply -f k8s/ticket-store.yaml
kubectl apply -f k8s/crew-store.yaml

# MCP servers
kubectl apply -f k8s/kb-mcp.yaml
kubectl apply -f k8s/ticket-mcp.yaml

# BFF
kubectl apply -f k8s/bff.yaml
kubectl apply -f k8s/bff-ingress.yaml

# Frontend
kubectl apply -f k8s/frontend.yaml

# agentgateway HTTPRoutes — attach services to the Solo Enterprise gateway.
# Requires agentgateway-proxy to exist in agentgateway-system (Phase 3).
# Safe to apply even if the gateway isn't installed yet — routes will attach
# once the gateway appears.
kubectl apply -f k8s/agentgateway-routes.yaml

echo ""
echo "==> Waiting for deployments to be ready..."
kubectl rollout status deployment/kb-store     -n amss --timeout=120s
kubectl rollout status deployment/ticket-store -n amss --timeout=120s
kubectl rollout status deployment/crew-store   -n amss --timeout=120s
kubectl rollout status deployment/kb-mcp       -n amss --timeout=120s
kubectl rollout status deployment/ticket-mcp   -n amss --timeout=120s
kubectl rollout status deployment/bff          -n amss --timeout=120s
kubectl rollout status deployment/frontend     -n amss --timeout=120s

echo ""
echo "==> All deployments ready."
echo ""
echo "  To access the application (through agentgateway):"
echo "    kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &"
echo "    open http://localhost:8080"
echo ""
echo "  To access the Solo Enterprise UI:"
echo "    kubectl port-forward svc/solo-enterprise-ui -n kagent 4000:80 &"
echo "    open http://localhost:4000"
echo ""
echo "  Debug (direct NodePort, bypasses agentgateway — not recommended):"
echo "    BFF API only:  http://localhost:30080"
echo "    Frontend only: http://localhost:30081 (API calls won't work)"
