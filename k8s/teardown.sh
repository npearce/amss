#!/usr/bin/env bash
# teardown.sh — Delete all AMSS resources from the cluster.
# Run from the repo root: ./k8s/teardown.sh
set -euo pipefail

echo "==> Deleting amss namespace (all resources inside will be removed)..."
kubectl delete namespace amss --ignore-not-found

echo "==> Done."
