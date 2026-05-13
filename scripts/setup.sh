#!/usr/bin/env bash
# setup.sh — One-command install of the full AMSS stack on a clean cluster.
# Run from the repo root: ./scripts/setup.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# ─────────────────────────────────────────────
# Step 0: Check required environment variables
# ─────────────────────────────────────────────
echo "==> Step 0: Checking required environment variables..."

missing=0
if [[ -z "${AGENTGATEWAY_LICENSE_KEY:-}" ]]; then
  echo "  ERROR: AGENTGATEWAY_LICENSE_KEY is not set"
  missing=1
fi
if [[ -z "${ANTHROPIC_API_KEY:-}" ]]; then
  echo "  ERROR: ANTHROPIC_API_KEY is not set"
  missing=1
fi
if [[ -z "${SOLO_ISTIO_LICENSE_KEY:-}" ]]; then
  echo "  ERROR: SOLO_ISTIO_LICENSE_KEY is not set"
  missing=1
fi

if [[ $missing -eq 1 ]]; then
  echo ""
  echo "  Obtain license keys from your Solo account representative."
  echo "  Then run:"
  echo "    export AGENTGATEWAY_LICENSE_KEY=<key>"
  echo "    export SOLO_ISTIO_LICENSE_KEY=<key>"
  echo "    export ANTHROPIC_API_KEY=<key>"
  exit 1
fi

echo "  All required environment variables are set."

# ─────────────────────────────────────────────
# Step 1: Set derived variables
# ─────────────────────────────────────────────
echo ""
echo "==> Step 1: Setting derived variables..."

KAGENT_ENT_VERSION=0.4.0
AGENTGATEWAY_VERSION=v2.3.2
ISTIO_VERSION=1.29.1
ISTIO_IMAGE="${ISTIO_VERSION}-solo"
ISTIO_REPO="us-docker.pkg.dev/solo-public/istio-helm"
MGMT_CONTEXT="$(kubectl config current-context)"

echo "  KAGENT_ENT_VERSION   = ${KAGENT_ENT_VERSION}"
echo "  AGENTGATEWAY_VERSION = ${AGENTGATEWAY_VERSION}"
echo "  ISTIO_VERSION        = ${ISTIO_VERSION}"
echo "  MGMT_CONTEXT         = ${MGMT_CONTEXT}"

# ─────────────────────────────────────────────
# Step 2: Install Gateway API CRDs
# ─────────────────────────────────────────────
echo ""
echo "==> Step 2: Installing Gateway API CRDs..."

kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.0/standard-install.yaml

echo "  Gateway API CRDs installed."

# ─────────────────────────────────────────────
# Step 3: Install Solo Enterprise agentgateway
# ─────────────────────────────────────────────
echo ""
echo "==> Step 3: Installing Solo Enterprise agentgateway..."

helm upgrade -i enterprise-agentgateway-crds \
  oci://us-docker.pkg.dev/solo-public/enterprise-agentgateway/charts/enterprise-agentgateway-crds \
  --create-namespace \
  --namespace agentgateway-system \
  --version "${AGENTGATEWAY_VERSION}"

helm upgrade -i enterprise-agentgateway \
  oci://us-docker.pkg.dev/solo-public/enterprise-agentgateway/charts/enterprise-agentgateway \
  -n agentgateway-system \
  --version "${AGENTGATEWAY_VERSION}" \
  --set-string licensing.licenseKey="${AGENTGATEWAY_LICENSE_KEY}"

echo "  Creating agentgateway Gateway resource..."
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: agentgateway-proxy
  namespace: agentgateway-system
spec:
  gatewayClassName: enterprise-agentgateway
  listeners:
  - protocol: HTTP
    port: 80
    name: http
    allowedRoutes:
      namespaces:
        from: All
EOF

echo "  agentgateway installed."

# ─────────────────────────────────────────────
# Step 4: Install kagent management chart
# ─────────────────────────────────────────────
echo ""
echo "==> Step 4: Installing kagent management chart..."

helm upgrade -i kagent-mgmt \
  oci://us-docker.pkg.dev/solo-public/solo-enterprise-helm/charts/management \
  --namespace kagent \
  --create-namespace \
  --version "${KAGENT_ENT_VERSION}" \
  --set cluster="mgmt-cluster" \
  --set products.kagent.enabled=true \
  --set products.agentgateway.enabled=true \
  --set-string licensing.licenseKey="${AGENTGATEWAY_LICENSE_KEY}"

echo "  kagent management chart installed."

# ─────────────────────────────────────────────
# Step 5: Install kagent CRDs + controller
# ─────────────────────────────────────────────
echo ""
echo "==> Step 5: Installing kagent CRDs and controller..."

helm upgrade -i kagent-crds \
  oci://us-docker.pkg.dev/solo-public/kagent-enterprise-helm/charts/kagent-enterprise-crds \
  --kube-context "${MGMT_CONTEXT}" \
  -n kagent --create-namespace \
  --version "${KAGENT_ENT_VERSION}"

echo "  Creating JWT secret..."
openssl genrsa -out /tmp/kagent-key.pem 2048
kubectl create secret generic jwt \
  -n kagent \
  --from-file=jwt=/tmp/kagent-key.pem \
  --dry-run=client -o yaml | kubectl apply -f -
rm -f /tmp/kagent-key.pem

echo "  Creating kagent values file..."
cat > /tmp/kagent-values.yaml <<EOF
licensing:
  licenseKey: ${AGENTGATEWAY_LICENSE_KEY}
providers:
  default: anthropic
  anthropic:
    apiKey: ${ANTHROPIC_API_KEY}
proxy:
  url: "http://agentgateway-proxy.agentgateway-system.svc.cluster.local:80"
otel:
  tracing:
    enabled: true
    exporter:
      otlp:
        endpoint: solo-enterprise-telemetry-collector.kagent.svc.cluster.local:4317
        insecure: true
EOF

helm upgrade -i kagent \
  oci://us-docker.pkg.dev/solo-public/kagent-enterprise-helm/charts/kagent-enterprise \
  -n kagent \
  --version "${KAGENT_ENT_VERSION}" \
  --values /tmp/kagent-values.yaml

rm -f /tmp/kagent-values.yaml

echo "  kagent controller installed."

# ─────────────────────────────────────────────
# Step 6: Install Solo distribution of Istio (ambient mode)
# ─────────────────────────────────────────────
echo ""
echo "==> Step 6: Installing Solo distribution of Istio (ambient mode)..."

helm upgrade -i istio-base \
  "oci://${ISTIO_REPO}/base" \
  --version "${ISTIO_VERSION}" \
  -n istio-system --create-namespace

helm upgrade -i istiod \
  "oci://${ISTIO_REPO}/istiod" \
  --version "${ISTIO_VERSION}" \
  -n istio-system \
  --set profile=ambient \
  --set-string global.hub="us-docker.pkg.dev/solo-public/istio" \
  --set-string global.tag="${ISTIO_IMAGE}" \
  --set-string licenseKey="${SOLO_ISTIO_LICENSE_KEY}" \
  --wait

helm upgrade -i istio-cni \
  "oci://${ISTIO_REPO}/cni" \
  --version "${ISTIO_VERSION}" \
  -n istio-system \
  --set profile=ambient \
  --set-string global.hub="us-docker.pkg.dev/solo-public/istio" \
  --set-string global.tag="${ISTIO_IMAGE}"

helm upgrade -i ztunnel \
  "oci://${ISTIO_REPO}/ztunnel" \
  --version "${ISTIO_VERSION}" \
  -n istio-system \
  --set-string global.hub="us-docker.pkg.dev/solo-public/istio" \
  --set-string global.tag="${ISTIO_IMAGE}"

echo "  Solo distribution of Istio installed."

# ─────────────────────────────────────────────
# Step 7: Build and deploy AMSS application
# ─────────────────────────────────────────────
echo ""
echo "==> Step 7: Building and deploying AMSS application..."

./k8s/deploy.sh

echo "  AMSS application deployed."

# ─────────────────────────────────────────────
# Step 8: Label amss namespace for ambient mesh
# ─────────────────────────────────────────────
echo ""
echo "==> Step 8: Enrolling amss namespace in ambient mesh..."

kubectl label namespace amss istio.io/dataplane-mode=ambient --overwrite

echo "  amss namespace enrolled."

# ─────────────────────────────────────────────
# Step 9: Configure LLM egress through agentgateway
# ─────────────────────────────────────────────
echo ""
echo "==> Step 9: Configuring LLM egress through agentgateway..."

kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: anthropic-secret
  namespace: agentgateway-system
type: Opaque
stringData:
  Authorization: ${ANTHROPIC_API_KEY}
EOF

kubectl apply -f - <<'EOF'
apiVersion: agentgateway.dev/v1alpha1
kind: AgentgatewayBackend
metadata:
  name: anthropic
  namespace: agentgateway-system
spec:
  ai:
    provider:
      anthropic:
        model: "claude-sonnet-4-6"
  policies:
    auth:
      secretRef:
        name: anthropic-secret
EOF

kubectl apply -f - <<'EOF'
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: anthropic
  namespace: agentgateway-system
spec:
  parentRefs:
  - name: agentgateway-proxy
    namespace: agentgateway-system
  rules:
  - backendRefs:
    - name: anthropic
      namespace: agentgateway-system
      group: agentgateway.dev
      kind: AgentgatewayBackend
EOF

echo "  LLM egress configured."

# ─────────────────────────────────────────────
# Step 10: Apply agentgateway tracing policy
# ─────────────────────────────────────────────
echo ""
echo "==> Step 10: Applying agentgateway tracing policy..."

kubectl apply -f - <<'EOF'
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: agentgateway-to-telemetry
  namespace: kagent
spec:
  from:
  - group: enterpriseagentgateway.solo.io
    kind: EnterpriseAgentgatewayPolicy
    namespace: agentgateway-system
  to:
  - group: ""
    kind: Service
---
apiVersion: enterpriseagentgateway.solo.io/v1alpha1
kind: EnterpriseAgentgatewayPolicy
metadata:
  name: tracing
  namespace: agentgateway-system
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: Gateway
    name: agentgateway-proxy
  frontend:
    tracing:
      backendRef:
        name: solo-enterprise-telemetry-collector
        namespace: kagent
        kind: Service
        port: 4317
      randomSampling: "true"
EOF

echo "  Tracing policy applied."

# ─────────────────────────────────────────────
# Step 11: Apply AMSS HTTPRoutes
# ─────────────────────────────────────────────
echo ""
echo "==> Step 11: Applying AMSS HTTPRoutes..."

kubectl apply -f k8s/agentgateway-routes.yaml

kubectl get httproute -n amss -o wide

echo "  HTTPRoutes applied."

# ─────────────────────────────────────────────
# Step 12: Create ModelConfig, RemoteMCPServers, Agent CRDs
# ─────────────────────────────────────────────
echo ""
echo "==> Step 12: Creating ModelConfig, RemoteMCPServers, and Agent CRDs..."

kubectl apply -f - <<'EOF'
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: default-model-config
  namespace: kagent
spec:
  provider: Anthropic
  model: claude-sonnet-4-6
EOF

kubectl apply -f - <<'EOF'
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: kb-mcp
  namespace: amss
spec:
  description: "Knowledge Base MCP server"
  url: http://kb-mcp.amss.svc.cluster.local:9001
  protocol: STREAMABLE_HTTP
  allowedNamespaces:
    from: All
EOF

kubectl apply -f - <<'EOF'
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: ticket-mcp
  namespace: amss
spec:
  description: "Ticket MCP server"
  url: http://ticket-mcp.amss.svc.cluster.local:9002
  protocol: STREAMABLE_HTTP
  allowedNamespaces:
    from: All
EOF

kubectl apply -f agents/mission-support-agent/agent.yaml
kubectl apply -f agents/kb-curator-agent/agent.yaml

echo "  ModelConfig, RemoteMCPServers, and Agent CRDs created."

# ─────────────────────────────────────────────
# Step 13: Enable live agent mode on BFF
# ─────────────────────────────────────────────
echo ""
echo "==> Step 13: Setting BFF to live agent mode (STUB_MODE=false)..."

kubectl set env deployment/bff -n amss \
  STUB_MODE=false \
  MISSION_SUPPORT_AGENT_URL=http://kagent-controller.kagent.svc.cluster.local:8083 \
  KAGENT_AGENT_NAMESPACE=kagent

echo "  Waiting for BFF to roll out..."
kubectl rollout status deployment/bff -n amss --timeout=120s

echo "  BFF is in live agent mode."

# ─────────────────────────────────────────────
# Step 14: Wait for all deployments to be ready
# ─────────────────────────────────────────────
echo ""
echo "==> Step 14: Waiting for all AMSS deployments to be ready..."

kubectl rollout status deployment/kb-store     -n amss --timeout=120s
kubectl rollout status deployment/ticket-store -n amss --timeout=120s
kubectl rollout status deployment/crew-store   -n amss --timeout=120s
kubectl rollout status deployment/kb-mcp       -n amss --timeout=120s
kubectl rollout status deployment/ticket-mcp   -n amss --timeout=120s
kubectl rollout status deployment/bff          -n amss --timeout=120s
kubectl rollout status deployment/frontend     -n amss --timeout=120s

echo "  All deployments are ready."

# ─────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────
echo ""
echo "=============================================="
echo "  AMSS Setup Complete — Full Stack"
echo "=============================================="
echo ""
echo "  Start access (run these once, leave running):"
echo ""
echo "    kubectl port-forward deployment/agentgateway-proxy -n agentgateway-system 8080:80 &"
echo "    kubectl port-forward svc/solo-enterprise-ui -n kagent 4000:80 &"
echo ""
echo "  Access points:"
echo "    AMSS Application:     http://localhost:8080"
echo "    Solo Enterprise UI:   http://localhost:4000"
echo ""
echo "  Quick smoke test:"
echo "    curl -s http://localhost:8080/health | jq .data.status"
echo "    curl -s http://localhost:8080/api/v1/crew | jq .data.total"
echo ""
echo "  Run a demo track:"
echo "    ./scripts/demo-track1.sh   # agentgateway only"
echo "    ./scripts/demo-track2.sh   # + kagent agents"
echo "    ./scripts/demo-track3.sh   # + ambient mesh"
echo "    ./scripts/demo-track4.sh   # full stack"
echo ""
