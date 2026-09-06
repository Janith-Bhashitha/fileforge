#!/bin/bash
set -euxo pipefail

# Runs once, on first boot. Output lands in /var/log/cloud-init-output.log,
# which is the first place to look if the site doesn't come up.
#
# This box never builds a Docker image - it only pulls the six that CI
# already built and pushed to Docker Hub. Building on a 1 GB instance is
# what used to freeze it: four concurrent Go compiles drove the load average
# past 17 and consumed the entire swapfile, locking the box up for hours
# with no OOM kill to force a recovery. Pulling is network-bound, not
# CPU/memory-bound, so that failure mode doesn't exist here any more.

# ---------------------------------------------------------------------------
# Swap. Kept even though builds are gone: Postgres, Redis, the API and five
# workers still add up over the course of normal use, and 1 GB leaves little
# margin. This is a safety net now, not load-bearing infrastructure.
# ---------------------------------------------------------------------------
dd if=/dev/zero of=/swapfile bs=1M count=2048
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
echo '/swapfile none swap sw 0 0' >> /etc/fstab
sysctl -w vm.swappiness=10
echo 'vm.swappiness=10' >> /etc/sysctl.conf

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------
dnf update -y
dnf install -y docker git

systemctl enable --now docker
usermod -aG docker ec2-user

# The compose plugin isn't in the AL2023 repos - the distro's docker package
# is the engine and CLI only. Pinned rather than "latest" so a boot six
# months from now provisions what this was tested against; needs 2.24+ for
# the !override tag the compose files use. buildx is deliberately NOT
# installed - nothing on this box ever builds an image any more.
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL "https://github.com/docker/compose/releases/download/v5.5.1/docker-compose-linux-x86_64" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
docker compose version

# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------
APP_DIR=/opt/fileforge
git clone --branch ${repo_branch} --depth 1 ${repo_url} "$APP_DIR"
cd "$APP_DIR"

# Storage points at S3, reached through the instance profile - no access keys
# are written anywhere on this box. DOCKERHUB_USERNAME is what makes the
# `image:` names in docker-compose.yml resolve to real, pullable images
# instead of the local-build-only default.
cat > "$APP_DIR/.env" <<ENVEOF
POSTGRES_USER=fileforge
POSTGRES_PASSWORD=${postgres_password}
POSTGRES_DB=fileforge
DATABASE_URL=postgres://fileforge:${postgres_password}@postgres:5432/fileforge?sslmode=disable
API_PORT=8080
JWT_SECRET=${jwt_secret}
REDIS_URL=redis://redis:6379
STORAGE_BACKEND=s3
S3_BUCKET=${s3_bucket}
S3_REGION=${region}
STORAGE_DIR=/data/storage
RATE_LIMIT_PER_MINUTE=120
MAX_CONCURRENT_JOBS=20
RETENTION_DAYS=7
DOCKERHUB_USERNAME=${dockerhub_username}
GEMINI_API_KEY=${gemini_api_key}
GEMINI_MODEL=${gemini_model}
ENVEOF
chmod 600 "$APP_DIR/.env"

# nginx (the web service) takes port 80 and proxies /api through to the API
# internally, so the API itself publishes nothing - one public entry point
# rather than the browser talking to two different ports.
#
# Postgres and Redis are bound to loopback rather than unpublished entirely:
# the migration step below connects from the host, but nothing outside the
# instance can reach either of them.
cat > "$APP_DIR/docker-compose.ec2.yml" <<'COMPOSEEOF'
services:
  web:
    ports: !override
      - "80:80"
      - "443:443"
  api:
    ports: !override []
  postgres:
    ports: !override
      - "127.0.0.1:5433:5432"
  redis:
    ports: !override
      - "127.0.0.1:6379:6379"
COMPOSEEOF

COMPOSE="docker compose -f docker-compose.yml -f docker-compose.ec2.yml"

# One network operation, not four CPU-bound ones. This is the entire fix.
$COMPOSE pull

$COMPOSE up -d postgres redis

curl -sSL https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-amd64.tar.gz \
  | tar xvz -C /usr/local/bin migrate
chmod +x /usr/local/bin/migrate

# Postgres accepts connections a moment after the container starts; retry
# rather than sleeping a fixed guess.
for i in $(seq 1 30); do
  if docker exec "$(docker ps -qf name=postgres)" pg_isready -U fileforge; then break; fi
  sleep 2
done

migrate -path "$APP_DIR/services/api/migrations" \
  -database "postgres://fileforge:${postgres_password}@localhost:5433/fileforge?sslmode=disable" up

$COMPOSE up -d

# ---------------------------------------------------------------------------
# Restart on reboot, and a nightly cleanup sweep.
# ---------------------------------------------------------------------------
cat > /etc/systemd/system/fileforge.service <<'SVCEOF'
[Unit]
Description=FileForge
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/fileforge
ExecStart=/usr/bin/docker compose -f docker-compose.yml -f docker-compose.ec2.yml up -d
ExecStop=/usr/bin/docker compose -f docker-compose.yml -f docker-compose.ec2.yml down

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl enable fileforge.service

# Retention sweep: without this, S3 grows until it leaves the 5 GB free tier.
cat > /etc/cron.daily/fileforge-cleanup <<'CRONEOF'
#!/bin/bash
cd /opt/fileforge && docker compose -f docker-compose.yml -f docker-compose.ec2.yml run --rm api /bin/cleanup
CRONEOF
chmod +x /etc/cron.daily/fileforge-cleanup

# Nightly redeploy: pulls whatever CI most recently pushed to main and
# restarts anything that changed. `up -d` only recreates containers whose
# image actually changed, so most nights this is a no-op.
cat > /etc/cron.daily/fileforge-redeploy <<'REDEOF'
#!/bin/bash
cd /opt/fileforge
git pull --ff-only
docker compose -f docker-compose.yml -f docker-compose.ec2.yml pull
docker compose -f docker-compose.yml -f docker-compose.ec2.yml up -d
docker image prune -f
REDEOF
chmod +x /etc/cron.daily/fileforge-redeploy

echo "FileForge boot provisioning finished"
