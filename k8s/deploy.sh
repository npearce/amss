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

# --- kind users: uncomment to load images into the cluster ---
# kind load docker-image amss/kb-store:latest
# kind load docker-image amss/ticket-store:latest
# kind load docker-image amss/crew-store:latest
# kind load docker-image amss/kb-mcp:latest
# kind load docker-image amss/ticket-mcp:latest
# kind load docker-image amss/bff:latest

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

echo ""
echo "==> Waiting for deployments to be ready..."
kubectl rollout status deployment/kb-store     -n amss --timeout=120s
kubectl rollout status deployment/ticket-store -n amss --timeout=120s
kubectl rollout status deployment/crew-store   -n amss --timeout=120s
kubectl rollout status deployment/kb-mcp       -n amss --timeout=120s
kubectl rollout status deployment/ticket-mcp   -n amss --timeout=120s
kubectl rollout status deployment/bff          -n amss --timeout=120s

echo ""
echo "==> All deployments ready."
echo ""
echo "Access the BFF:"
echo "  NodePort (OrbStack):  http://localhost:30080"
echo "  Port-forward:         kubectl port-forward svc/bff 8080:8080 -n amss"
echo ""
echo "Quick health check:"
echo "  curl http://localhost:30080/health"
echo "  # or after port-forward:"
echo "  curl http://localhost:8080/health"
