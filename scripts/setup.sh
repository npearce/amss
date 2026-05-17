#!/usr/bin/env bash
# setup.sh — One-command install of the full AMSS stack on a clean cluster.
# Run from the repo root: ./scripts/setup.sh [--with-mesh]
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# ─────────────────────────────────────────────
# Flag parsing
# ─────────────────────────────────────────────
INSTALL_MESH=false
for arg in "$@"; do
  case $arg in
    --with-mesh) INSTALL_MESH=true ;;
    *) echo "Unknown option: $arg"; echo "Usage: ./scripts/setup.sh [--with-mesh]"; exit 1 ;;
  esac
done

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
if [ "$INSTALL_MESH" = true ] && [[ -z "${SOLO_ISTIO_LICENSE_KEY:-}" ]]; then
  echo "  ERROR: SOLO_ISTIO_LICENSE_KEY is not set (required for --with-mesh)"
  missing=1
fi

if [[ $missing -eq 1 ]]; then
  echo ""
  echo "  Obtain license keys from your Solo account representative."
  echo "  Then run:"
  echo "    export AGENTGATEWAY_LICENSE_KEY=<key>"
  echo "    export ANTHROPIC_API_KEY=<key>"
  if [ "$INSTALL_MESH" = true ]; then
    echo "    export SOLO_ISTIO_LICENSE_KEY=<key>   # required for --with-mesh"
  fi
  exit 1
fi

# KAGENT_LICENSE_KEY defaults to AGENTGATEWAY_LICENSE_KEY in demo environments
# (both keys are issued together; they share the same value for most installs).
KAGENT_LICENSE_KEY="${KAGENT_LICENSE_KEY:-${AGENTGATEWAY_LICENSE_KEY}}"

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
ISTIO_HUB="us-docker.pkg.dev/soloio-img/istio"
ISTIO_REPO="us-docker.pkg.dev/soloio-img/istio-helm"
MGMT_CONTEXT="$(kubectl config current-context)"

echo "  KAGENT_ENT_VERSION   = ${KAGENT_ENT_VERSION}"
echo "  AGENTGATEWAY_VERSION = ${AGENTGATEWAY_VERSION}"
echo "  INSTALL_MESH         = ${INSTALL_MESH}"
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
# Step 4: Install kagent CRDs
# (must be before the management chart so the UI sees them on startup)
# ─────────────────────────────────────────────
echo ""
echo "==> Step 4: Installing kagent CRDs..."

helm upgrade -i kagent-crds \
  oci://us-docker.pkg.dev/solo-public/kagent-enterprise-helm/charts/kagent-enterprise-crds \
  --kube-context "${MGMT_CONTEXT}" \
  -n kagent --create-namespace \
  --version "${KAGENT_ENT_VERSION}"

echo "  kagent CRDs installed."

# ─────────────────────────────────────────────
# Step 5: Install kagent management chart + register cluster
# ─────────────────────────────────────────────
echo ""
echo "==> Step 5: Installing kagent management chart..."

helm upgrade -i kagent-mgmt \
  oci://us-docker.pkg.dev/solo-public/solo-enterprise-helm/charts/management \
  --namespace kagent \
  --create-namespace \
  --version "${KAGENT_ENT_VERSION}" \
  --set cluster="mgmt-cluster" \
  --set products.kagent.enabled=true \
  --set products.agentgateway.enabled=true \
  --set products.agentgateway.namespace=agentgateway-system \
  --set-string licensing.licenseKey="${AGENTGATEWAY_LICENSE_KEY}" \
  --no-hooks

echo "  Registering KubernetesCluster so the UI can discover the cluster..."
kubectl apply -f - <<'EOF'
apiVersion: platform.solo.io/v1alpha1
kind: KubernetesCluster
metadata:
  name: mgmt-cluster
  namespace: kagent
EOF

echo "  kagent management chart installed."

# ─────────────────────────────────────────────
# Step 6: Create JWT secret and install kagent controller
# ─────────────────────────────────────────────
echo ""
echo "==> Step 6: Installing kagent controller..."

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
  --set oidc.skipOBO=true \
  --set kmcp.licensing.createSecret=false \
  --values /tmp/kagent-values.yaml

rm -f /tmp/kagent-values.yaml

echo "  Fixing OTEL tracing endpoint format..."
kubectl set env deployment/kagent-controller -n kagent \
  OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://solo-enterprise-telemetry-collector.kagent.svc.cluster.local:4317
kubectl rollout status deployment/kagent-controller -n kagent --timeout=120s

echo "  kagent controller installed."

# ─────────────────────────────────────────────
# Step 7: Build and deploy AMSS application
# ─────────────────────────────────────────────
echo ""
echo "==> Step 7: Building and deploying AMSS application..."

./k8s/deploy.sh

echo "  AMSS application deployed."

# ─────────────────────────────────────────────
# Step 8 (optional): Install Solo distribution of Istio + enroll amss
# ─────────────────────────────────────────────
echo ""
if [ "$INSTALL_MESH" = true ]; then
  echo "==> Step 8: Installing Solo distribution of Istio (ambient mode)..."

  echo "  Cleaning up any stale validating webhooks to prevent field ownership conflicts..."
  kubectl delete validatingwebhookconfiguration istiod-default-validator 2>/dev/null || true
  kubectl delete validatingwebhookconfiguration istio-validator-istio-system 2>/dev/null || true

  helm upgrade -i istio-base \
    "oci://${ISTIO_REPO}/base" \
    --version "${ISTIO_VERSION}" \
    -n istio-system --create-namespace

  echo "  Applying istiod Segments RBAC (required for Solo distribution)..."
  kubectl apply -f - <<'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: istiod-segments
rules:
- apiGroups: ["admin.solo.io"]
  resources: ["segments"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: istiod-segments
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: istiod-segments
subjects:
- kind: ServiceAccount
  name: istiod
  namespace: istio-system
EOF

  helm upgrade -i istiod \
    "oci://${ISTIO_REPO}/istiod" \
    --version "${ISTIO_VERSION}" \
    -n istio-system \
    --set profile=ambient \
    --set-string global.hub="${ISTIO_HUB}" \
    --set-string global.tag="${ISTIO_IMAGE}" \
    --set-string licenseKey="${SOLO_ISTIO_LICENSE_KEY}" \
    --timeout 5m \
    --wait

  helm upgrade -i istio-cni \
    "oci://${ISTIO_REPO}/cni" \
    --version "${ISTIO_VERSION}" \
    -n istio-system \
    --set profile=ambient \
    --set-string global.hub="${ISTIO_HUB}" \
    --set-string global.tag="${ISTIO_IMAGE}"

  helm upgrade -i ztunnel \
    "oci://${ISTIO_REPO}/ztunnel" \
    --version "${ISTIO_VERSION}" \
    -n istio-system \
    --set-string global.hub="${ISTIO_HUB}" \
    --set-string global.tag="${ISTIO_IMAGE}"

  echo "  Enrolling amss namespace in ambient mesh..."
  # Only amss is enrolled. kagent excluded (ztunnel breaks outbound HTTPS to LLM
  # providers). agentgateway-system excluded (gateway manages its own TLS).
  kubectl label namespace amss istio.io/dataplane-mode=ambient --overwrite

  echo "  Solo distribution of Istio installed. amss namespace enrolled."
else
  echo "==> Step 8: Skipping ambient mesh (use --with-mesh to include)."
fi

# ─────────────────────────────────────────────
# Step 9: Configure LLM egress through agentgateway
# ─────────────────────────────────────────────
echo ""
echo "==> Step 9: Configuring LLM egress through agentgateway..."

# LLM Egress Architecture:
# - kagent agents speak OpenAI format (provider: OpenAI in ModelConfig)
# - Requests route to agentgateway via ModelConfig openAI.baseUrl (/anthropic path)
# - agentgateway translates OpenAI format → Anthropic native format
# - agentgateway injects the real Anthropic API key from its own secret
# - agentgateway forwards to api.anthropic.com with TLS
# - agentgateway parses Anthropic responses for token usage metrics
#
# This approach:
# 1. Gives full LLM observability (token counts, latency) in agentgateway UI
# 2. Centralizes API key management in agentgateway
# 3. Enables provider switching by changing one AgentgatewayBackend — zero agent changes
#
# Note: Direct Anthropic AI backend has a known bug (agentgateway-enterprise#520)
# where Anthropic-native tool definitions are rejected. The OpenAI→Anthropic
# translation path works around this.

# 1. Anthropic API key secret — agentgateway uses this for auth injection
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

# 2. AI backend — agentgateway translates OpenAI→Anthropic and injects the API key
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

# 3. HTTPRoute — /anthropic path prefix for LLM traffic
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
  - matches:
    - path:
        type: PathPrefix
        value: /anthropic
    backendRefs:
    - name: anthropic
      namespace: agentgateway-system
      group: agentgateway.dev
      kind: AgentgatewayBackend
EOF

# 4. Dummy OpenAI key — kagent's SDK requires a key for initialization even
#    though agentgateway handles the real auth. The ModelConfig points at
#    agentgateway, so this key is never sent to a real OpenAI endpoint.
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Secret
metadata:
  name: kagent-openai-dummy
  namespace: kagent
type: Opaque
stringData:
  OPENAI_API_KEY: "sk-dummy-not-used-agentgateway-handles-auth"
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

echo "  Removing Helm-managed ModelConfig to avoid field ownership conflict..."
kubectl delete modelconfig default-model-config -n kagent 2>/dev/null || true

# ModelConfig uses OpenAI provider pointing at agentgateway's /anthropic path.
# agentgateway translates the OpenAI request format to Anthropic native format
# and injects the real API key. Workaround for agentgateway-enterprise#520.
kubectl apply -f - <<'EOF'
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: default-model-config
  namespace: kagent
spec:
  provider: OpenAI
  model: claude-sonnet-4-6
  apiKeySecret: kagent-openai-dummy
  apiKeySecretKey: OPENAI_API_KEY
  openAI:
    baseUrl: http://agentgateway-proxy.agentgateway-system.svc.cluster.local/anthropic
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
if [ "$INSTALL_MESH" = true ]; then
  echo "  AMSS Setup Complete — Full Stack (with ambient mesh)"
else
  echo "  AMSS Setup Complete — agentgateway + kagent"
fi
echo "=============================================="
echo ""
echo "  Start the demo:"
echo "    ./scripts/demo.sh"
echo ""
echo "  Access points:"
echo "    AMSS Application:     http://localhost:8080"
echo "    Solo Enterprise UI:   http://localhost:4000"
echo ""
echo "  Quick smoke test:"
echo "    curl -s http://localhost:8080/health | jq .data.status"
echo "    curl -s http://localhost:8080/api/v1/crew | jq .data.total"
echo ""
if [ "$INSTALL_MESH" = true ]; then
  echo "  Ambient mesh: amss namespace enrolled (mTLS on data layer)"
  echo "    kubectl get namespace amss --show-labels | grep istio"
  echo "    kubectl get pods -n amss   # all should be 1/1 (no sidecars)"
  echo ""
fi
