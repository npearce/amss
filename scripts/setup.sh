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
# Note: Direct Anthropic AI backend has a known bug (agentgateway-enterprise#520)
# where Anthropic-native tool definitions are rejected. The OpenAI→Anthropic
# translation path works around this.

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

# Dummy OpenAI key — kagent's SDK requires a key for initialization even
# though agentgateway handles the real auth.
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
# Step 12: Deploy and configure Keycloak
# ─────────────────────────────────────────────
echo ""
echo "==> Step 12: Deploying and configuring Keycloak..."

kubectl create namespace keycloak 2>/dev/null || true
kubectl -n keycloak apply -f https://raw.githubusercontent.com/solo-io/gloo-mesh-use-cases/main/policy-demo/oidc/keycloak.yaml
kubectl -n keycloak rollout status deploy/keycloak --timeout=120s

echo "  Waiting for Keycloak endpoint..."
KEYCLOAK_HOST=""
KEYCLOAK_PORT=""
for i in $(seq 1 30); do
  KEYCLOAK_HOST=$(kubectl -n keycloak get service keycloak -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null) || true
  if [ -n "$KEYCLOAK_HOST" ]; then break; fi
  sleep 2
done
if [ -z "$KEYCLOAK_HOST" ]; then
  KEYCLOAK_HOST="localhost"
  KEYCLOAK_PORT=$(kubectl -n keycloak get service keycloak -o jsonpath='{.spec.ports[0].nodePort}')
  echo "  No LoadBalancer IP — using NodePort: ${KEYCLOAK_HOST}:${KEYCLOAK_PORT}"
else
  KEYCLOAK_PORT=8080
fi
KEYCLOAK_URL="http://${KEYCLOAK_HOST}:${KEYCLOAK_PORT}"
echo "  Keycloak reachable at $KEYCLOAK_URL"

# Admin token helper — tokens expire in 60s, refresh before each batch of calls
get_keycloak_token() {
  curl -s -d "client_id=admin-cli" -d "username=admin" -d "password=admin" \
    -d "grant_type=password" \
    "$KEYCLOAK_URL/realms/master/protocol/openid-connect/token" | jq -r .access_token
}

echo "  Waiting for Keycloak to be ready..."
for i in $(seq 1 30); do
  if curl -s -o /dev/null -w "%{http_code}" "$KEYCLOAK_URL/realms/master" | grep -q "200"; then
    break
  fi
  sleep 2
done

KC_TOKEN=$(get_keycloak_token)
echo "  Admin token acquired."

echo "  Registering OIDC client..."
KEYCLOAK_CLIENT=""
kc_reg_token=""
read -r KEYCLOAK_CLIENT kc_reg_token <<<$(curl -s -H "Authorization: Bearer ${KC_TOKEN}" \
  -X POST -H "Content-Type: application/json" \
  -d '{"expiration": 0, "count": 1}' \
  "$KEYCLOAK_URL/admin/realms/master/clients-initial-access" | jq -r '[.id, .token] | @tsv')

kc_id=""
KEYCLOAK_SECRET=""
read -r kc_id KEYCLOAK_SECRET <<<$(curl -s -k -X POST \
  -d "{ \"clientId\": \"${KEYCLOAK_CLIENT}\" }" \
  -H "Content-Type:application/json" \
  -H "Authorization: bearer ${kc_reg_token}" \
  "${KEYCLOAK_URL}/realms/master/clients-registrations/default" | jq -r '[.id, .secret] | @tsv')

echo "  Client ID: $KEYCLOAK_CLIENT"

KC_TOKEN=$(get_keycloak_token)

echo "  Configuring client settings..."
curl -s -H "Authorization: Bearer ${KC_TOKEN}" -X PUT -H "Content-Type: application/json" \
  -d '{"serviceAccountsEnabled": true, "directAccessGrantsEnabled": true, "authorizationServicesEnabled": true, "redirectUris": ["*"]}' \
  "$KEYCLOAK_URL/admin/realms/master/clients/${kc_id}" > /dev/null

echo "  Adding JWT claim mappers..."
for mapper in \
  '{"name":"group","protocol":"openid-connect","protocolMapper":"oidc-usermodel-attribute-mapper","config":{"claim.name":"group","jsonType.label":"String","user.attribute":"group","id.token.claim":"true","access.token.claim":"true"}}' \
  '{"name":"crew_id","protocol":"openid-connect","protocolMapper":"oidc-usermodel-attribute-mapper","config":{"claim.name":"crew_id","jsonType.label":"String","user.attribute":"crew_id","id.token.claim":"true","access.token.claim":"true"}}' \
  '{"name":"mission","protocol":"openid-connect","protocolMapper":"oidc-usermodel-attribute-mapper","config":{"claim.name":"mission","jsonType.label":"String","user.attribute":"mission","id.token.claim":"true","access.token.claim":"true"}}'; do
  curl -s -H "Authorization: Bearer ${KC_TOKEN}" -X POST -H "Content-Type: application/json" \
    -d "$mapper" \
    "$KEYCLOAK_URL/admin/realms/master/clients/${kc_id}/protocol-mappers/models" > /dev/null
done

KC_TOKEN=$(get_keycloak_token)

echo "  Creating crew members..."
for user in \
  '{"username":"wiseman","email":"wiseman@artemis.nasa.gov","firstName":"Reid","lastName":"Wiseman","enabled":true,"attributes":{"group":"crew","crew_id":"wiseman-r","mission":"artemis-ii"},"credentials":[{"type":"password","value":"artemis","temporary":false}]}' \
  '{"username":"glover","email":"glover@artemis.nasa.gov","firstName":"Victor","lastName":"Glover","enabled":true,"attributes":{"group":"crew","crew_id":"glover-v","mission":"artemis-ii"},"credentials":[{"type":"password","value":"artemis","temporary":false}]}' \
  '{"username":"koch","email":"koch@artemis.nasa.gov","firstName":"Christina","lastName":"Koch","enabled":true,"attributes":{"group":"crew","crew_id":"koch-c","mission":"artemis-ii"},"credentials":[{"type":"password","value":"artemis","temporary":false}]}' \
  '{"username":"hansen","email":"hansen@artemis.nasa.gov","firstName":"Jeremy","lastName":"Hansen","enabled":true,"attributes":{"group":"crew","crew_id":"hansen-j","mission":"artemis-ii"},"credentials":[{"type":"password","value":"artemis","temporary":false}]}'; do
  result=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer ${KC_TOKEN}" \
    -X POST -H "Content-Type: application/json" -d "$user" \
    "$KEYCLOAK_URL/admin/realms/master/users")
  name=$(echo "$user" | jq -r .username)
  if [ "$result" = "201" ]; then echo "    Created: $name"
  elif [ "$result" = "409" ]; then echo "    Already exists: $name"
  else echo "    Failed ($result): $name"; fi
done

KC_TOKEN=$(get_keycloak_token)

echo "  Creating ground control users..."
for user in \
  '{"username":"gc-eclss","email":"eclss@mission-control.nasa.gov","firstName":"ECLSS","lastName":"Officer","enabled":true,"attributes":{"group":"ground-control","crew_id":"gc-eclss","mission":"artemis-ii"},"credentials":[{"type":"password","value":"houston","temporary":false}]}' \
  '{"username":"gc-comm","email":"comm@mission-control.nasa.gov","firstName":"COMM","lastName":"Officer","enabled":true,"attributes":{"group":"ground-control","crew_id":"gc-comm","mission":"artemis-ii"},"credentials":[{"type":"password","value":"houston","temporary":false}]}' \
  '{"username":"gc-eva","email":"eva@mission-control.nasa.gov","firstName":"EVA","lastName":"Officer","enabled":true,"attributes":{"group":"ground-control","crew_id":"gc-eva","mission":"artemis-ii"},"credentials":[{"type":"password","value":"houston","temporary":false}]}' \
  '{"username":"gc-gnc","email":"gnc@mission-control.nasa.gov","firstName":"GNC","lastName":"Officer","enabled":true,"attributes":{"group":"ground-control","crew_id":"gc-gnc","mission":"artemis-ii"},"credentials":[{"type":"password","value":"houston","temporary":false}]}' \
  '{"username":"amss-agent","email":"agent@amss.local","firstName":"AMSS","lastName":"Agent","enabled":true,"attributes":{"group":"service-account","crew_id":"amss-agent","mission":"artemis-ii"},"credentials":[{"type":"password","value":"agent-secret","temporary":false}]}'; do
  result=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer ${KC_TOKEN}" \
    -X POST -H "Content-Type: application/json" -d "$user" \
    "$KEYCLOAK_URL/admin/realms/master/users")
  name=$(echo "$user" | jq -r .username)
  if [ "$result" = "201" ]; then echo "    Created: $name"
  elif [ "$result" = "409" ]; then echo "    Already exists: $name"
  else echo "    Failed ($result): $name"; fi
done

KC_TOKEN=$(get_keycloak_token)

echo "  Removing client registration policies (testing only)..."
trusted_hosts=$(curl -s -H "Authorization: Bearer ${KC_TOKEN}" \
  "${KEYCLOAK_URL}/admin/realms/master/components?type=org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy" \
  | jq -r 'if type=="array" then .[] | select(.providerId=="trusted-hosts") | .id else empty end')
if [ -n "$trusted_hosts" ]; then
  curl -s -X DELETE -H "Authorization: Bearer ${KC_TOKEN}" \
    "${KEYCLOAK_URL}/admin/realms/master/components/${trusted_hosts}" > /dev/null
  echo "    Removed trusted-hosts policy"
fi

allowed_templates=$(curl -s -H "Authorization: Bearer ${KC_TOKEN}" \
  "${KEYCLOAK_URL}/admin/realms/master/components?type=org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy" \
  | jq -r '.[] | select(.providerId=="allowed-client-templates" and .subType=="anonymous") | .id')
if [ -n "$allowed_templates" ]; then
  curl -s -X DELETE -H "Authorization: Bearer ${KC_TOKEN}" \
    "${KEYCLOAK_URL}/admin/realms/master/components/${allowed_templates}" > /dev/null
  echo "    Removed allowed-client-templates policy"
fi

echo "  Setting token lifetime to 30 minutes..."
TOKEN=$(get_keycloak_token)
curl -s -H "Authorization: Bearer ${TOKEN}" -X PUT -H "Content-Type: application/json" \
  -d '{"accessTokenLifespan": 1800}' \
  "$KEYCLOAK_URL/admin/realms/master"

echo "  Creating AuthConfig for JWT validation..."
KEYCLOAK_CERT_KEYS=$(curl -s "$KEYCLOAK_URL/realms/master/protocol/openid-connect/certs" | jq -c .)

kubectl apply -f - <<AUTHEOF
apiVersion: extauth.solo.io/v1
kind: AuthConfig
metadata:
  name: keycloak-jwt
  namespace: agentgateway-system
spec:
  configs:
  - oauth2:
      accessTokenValidation:
        jwt:
          localJwks:
            inlineString: '${KEYCLOAK_CERT_KEYS}'
AUTHEOF

echo "  Applying auth policy to ingress routes..."
kubectl apply -f - <<'EOF'
apiVersion: enterpriseagentgateway.solo.io/v1alpha1
kind: EnterpriseAgentgatewayPolicy
metadata:
  name: keycloak-auth
  namespace: amss
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: bff-api-route
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: frontend-route
  traffic:
    entExtAuth:
      authConfigRef:
        name: keycloak-jwt
        namespace: agentgateway-system
      backendRef:
        name: ext-auth-service-enterprise-agentgateway
        namespace: agentgateway-system
        port: 8083
EOF

echo "  Saving client credentials to k8s secret..."
kubectl create secret generic keycloak-client -n amss \
  --from-literal=client-id="$KEYCLOAK_CLIENT" \
  --from-literal=client-secret="$KEYCLOAK_SECRET" \
  --from-literal=keycloak-url="$KEYCLOAK_URL" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "  Keycloak configured."

# ─────────────────────────────────────────────
# Step 13: Create ModelConfig, RemoteMCPServers, Agent CRDs
# ─────────────────────────────────────────────
echo ""
echo "==> Step 13: Creating ModelConfig, RemoteMCPServers, and Agent CRDs..."

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
# Step 14: Enable live agent mode on BFF
# ─────────────────────────────────────────────
echo ""
echo "==> Step 14: Setting BFF to live agent mode (STUB_MODE=false)..."

kubectl set env deployment/bff -n amss \
  STUB_MODE=false \
  MISSION_SUPPORT_AGENT_URL=http://kagent-controller.kagent.svc.cluster.local:8083 \
  KAGENT_AGENT_NAMESPACE=kagent

echo "  Waiting for BFF to roll out..."
kubectl rollout status deployment/bff -n amss --timeout=120s

echo "  BFF is in live agent mode."

# ─────────────────────────────────────────────
# Step 15: Wait for all deployments to be ready
# ─────────────────────────────────────────────
echo ""
echo "==> Step 15: Waiting for all AMSS deployments to be ready..."

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
echo "  Keycloak:"
echo "    URL:    $KEYCLOAK_URL"
echo "    Admin:  admin / admin"
echo "    Crew:   wiseman, glover, koch, hansen (password: artemis)"
echo "    GC:     gc-eclss, gc-comm, gc-eva, gc-gnc (password: houston)"
echo "    Agent:  amss-agent (password: agent-secret)"
echo ""
echo "  Get a user token:"
echo "    curl -s -d \"client_id=${KEYCLOAK_CLIENT}\" -d \"client_secret=${KEYCLOAK_SECRET}\" \\"
echo "      -d \"username=wiseman\" -d \"password=artemis\" -d \"grant_type=password\" \\"
echo "      \"${KEYCLOAK_URL}/realms/master/protocol/openid-connect/token\" | jq -r .access_token"
echo ""
if [ "$INSTALL_MESH" = true ]; then
  echo "  Ambient mesh: amss namespace enrolled (mTLS on data layer)"
  echo "    kubectl get namespace amss --show-labels | grep istio"
  echo "    kubectl get pods -n amss   # all should be 1/1 (no sidecars)"
  echo ""
fi
