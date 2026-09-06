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
COPY infrastructure/docker/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
