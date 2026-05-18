# ── Build stage ──────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

# VITE_API_URL can be injected at build time:
#   docker build --build-arg VITE_API_URL=http://bff.amss.svc.cluster.local:8080 .
# Leave empty to use the Vite proxy (dev only) or configure at runtime via env.
ARG VITE_API_URL=""
ENV VITE_API_URL=${VITE_API_URL}

# Keycloak ROPC auth — baked in at build time by deploy.sh.
# If these are empty, the frontend runs without auth (Keycloak not installed).
ARG VITE_KEYCLOAK_URL=""
ENV VITE_KEYCLOAK_URL=${VITE_KEYCLOAK_URL}

ARG VITE_KEYCLOAK_CLIENT_ID=""
ENV VITE_KEYCLOAK_CLIENT_ID=${VITE_KEYCLOAK_CLIENT_ID}

ARG VITE_KEYCLOAK_CLIENT_SECRET=""
ENV VITE_KEYCLOAK_CLIENT_SECRET=${VITE_KEYCLOAK_CLIENT_SECRET}

RUN npm run build

# ── Serve stage ──────────────────────────────────────────────
FROM nginx:1.25-alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
