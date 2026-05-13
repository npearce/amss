#!/usr/bin/env bash
# teardown.sh — Complete teardown of AMSS and all Solo Enterprise products.
# Run from the repo root: ./scripts/teardown.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=============================================="
echo "  AMSS Teardown"
echo "=============================================="
echo ""
echo "  This will remove all AMSS resources and Solo Enterprise products."
echo "  Press Ctrl+C within 5 seconds to abort."
echo ""
sleep 5

# ─────────────────────────────────────────────
# 1. Remove ambient mesh label from amss namespace
# ─────────────────────────────────────────────
echo "==> Removing ambient mesh label from amss namespace..."
kubectl label namespace amss istio.io/dataplane-mode- --ignore-not-found 2>/dev/null || true

# ─────────────────────────────────────────────
# 2. Delete AMSS application namespace
# ─────────────────────────────────────────────
echo "==> Deleting AMSS application namespace..."
kubectl delete namespace amss --ignore-not-found || true

# ─────────────────────────────────────────────
# 3. Uninstall ambient mesh (reverse install order)
# ─────────────────────────────────────────────
echo "==> Uninstalling ambient mesh (ztunnel, cni, istiod, base)..."
helm uninstall ztunnel    -n istio-system 2>/dev/null || true
helm uninstall istio-cni  -n istio-system 2>/dev/null || true
helm uninstall istiod     -n istio-system 2>/dev/null || true
helm uninstall istio-base -n istio-system 2>/dev/null || true

echo "==> Deleting istio-system namespace..."
kubectl delete namespace istio-system --ignore-not-found || true

# ─────────────────────────────────────────────
# 4. Uninstall kagent
# ─────────────────────────────────────────────
echo "==> Uninstalling kagent controller..."
helm uninstall kagent -n kagent 2>/dev/null || true

echo "==> Uninstalling kagent CRDs..."
helm uninstall kagent-crds -n kagent 2>/dev/null || true

echo "==> Uninstalling kagent management chart..."
helm uninstall kagent-mgmt -n kagent 2>/dev/null || true

echo "==> Deleting kagent namespace..."
kubectl delete namespace kagent --ignore-not-found || true

# ─────────────────────────────────────────────
# 5. Uninstall agentgateway
# ─────────────────────────────────────────────
echo "==> Uninstalling agentgateway control plane..."
helm uninstall enterprise-agentgateway -n agentgateway-system 2>/dev/null || true

echo "==> Uninstalling agentgateway CRDs..."
helm uninstall enterprise-agentgateway-crds -n agentgateway-system 2>/dev/null || true

echo "==> Deleting agentgateway-system namespace..."
kubectl delete namespace agentgateway-system --ignore-not-found || true

# ─────────────────────────────────────────────
# 6. Delete Gateway API CRDs
# ─────────────────────────────────────────────
echo "==> Deleting Gateway API CRDs..."
kubectl delete -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.0/standard-install.yaml \
  --ignore-not-found 2>/dev/null || true

echo ""
echo "=============================================="
echo "  Teardown complete."
echo "=============================================="
echo ""
echo "  Cluster is clean. To redeploy from scratch:"
echo "    ./scripts/setup.sh"
echo ""
