FROM node:22-alpine AS builder
WORKDIR /app

COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci

COPY apps/web/. .

# Built with an empty API base URL on purpose: every request then goes to the
# same origin as the page ("/api/v1/..."), and nginx proxies those through to
# the API container. That means no CORS, and no rebuild of the frontend when
# the backend moves - the alternative, baking an absolute URL in at build
# time, hardcodes the deployment's address into the bundle.
ENV VITE_API_BASE_URL=""
RUN npm run build

FROM nginx:1.27-alpine

# Self-signed, generated once at build time. There's no domain name to
# validate a real certificate against (this is served from a bare IP), so
# this is what stands between "HTTPS times out" and "HTTPS shows a one-time
# browser warning" - see nginx.conf for why that gap matters here.
RUN apk add --no-cache openssl \
    && mkdir -p /etc/nginx/ssl \
    && openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
         -keyout /etc/nginx/ssl/selfsigned.key \
         -out /etc/nginx/ssl/selfsigned.crt \
         -subj "/CN=fileforge"

COPY infrastructure/docker/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY infrastructure/docker/web/locations.conf /etc/nginx/locations.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80 443
