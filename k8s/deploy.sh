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
# No VITE_API_URL — the SPA uses /api/v1/... as a relative path by default.
# This works when frontend and BFF are behind the same host (agentgateway).
# To target a specific BFF host (e.g. direct NodePort access without agentgateway):
#   docker build --build-arg VITE_API_URL=http://localhost:30080 -t amss/frontend:latest ./frontend
docker build -t amss/frontend:latest ./frontend

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
# Requires amss-gateway to exist in agentgateway-system (Phase 3).
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
echo "  agentgateway:  http://192.168.139.2        (frontend + BFF via gateway)"
echo "  Frontend:      http://localhost:30081       (NodePort direct — dev/debug only)"
echo "  BFF API:       http://localhost:30080       (NodePort direct — dev/debug only)"
echo ""
echo "Quick health check:"
echo "  curl http://192.168.139.2/health            # via gateway"
echo "  curl http://localhost:30080/health           # direct NodePort"
