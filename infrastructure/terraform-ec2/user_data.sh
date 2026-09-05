#!/bin/bash
set -euxo pipefail

# Runs once, on first boot. Output lands in /var/log/cloud-init-output.log,
# which is the first place to look if the site doesn't come up.

# ---------------------------------------------------------------------------
# Swap. t3.micro has 1 GB of RAM and this stack runs Postgres, Redis, the API
# and three workers - one of which shells out to LibreOffice, briefly wanting
# a few hundred MB on its own. Without swap the kernel OOM-kills a container
# mid-conversion and the failure looks like a mysterious worker restart.
# 2 GB of swap on the root volume is the cheap fix; it is slower than RAM but
# only the idle containers get paged out.
# ---------------------------------------------------------------------------
dd if=/dev/zero of=/swapfile bs=1M count=2048
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
echo '/swapfile none swap sw 0 0' >> /etc/fstab

# Prefer reclaiming page cache over swapping processes out.
sysctl -w vm.swappiness=10
echo 'vm.swappiness=10' >> /etc/sysctl.conf

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------
dnf update -y
dnf install -y docker git

systemctl enable --now docker
usermod -aG docker ec2-user

# Neither the compose nor the buildx plugin is in the AL2023 repos - the
# distro's docker package is the engine and CLI only. Both are installed
# where Docker looks for plugins.
#
# buildx is not optional here: `docker compose build` refuses to run without
# buildx 0.17+, so leaving it out fails the whole provision at the build step.
#
# Both are pinned rather than "latest" so a boot six months from now
# provisions what this was tested against. Compose needs 2.24+ for the
# !override tag the compose files use.
mkdir -p /usr/local/lib/docker/cli-plugins

curl -SL "https://github.com/docker/buildx/releases/download/v0.37.0/buildx-v0.37.0.linux-amd64" \
  -o /usr/local/lib/docker/cli-plugins/docker-buildx
chmod +x /usr/local/lib/docker/cli-plugins/docker-buildx

curl -SL "https://github.com/docker/compose/releases/download/v5.5.1/docker-compose-linux-x86_64" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

docker buildx version
docker compose version

# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------
APP_DIR=/opt/fileforge
git clone --branch ${repo_branch} --depth 1 ${repo_url} "$APP_DIR"
cd "$APP_DIR"

# Storage points at S3, reached through the instance profile - no access keys
# are written anywhere on this box.
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
ENVEOF
chmod 600 "$APP_DIR/.env"

# Serve the API on port 80 so the instance is reachable without a port in
# the URL, and without paying for a load balancer to do it.
# Postgres and Redis are bound to loopback rather than unpublished entirely:
# the migration step below connects from the host, but nothing outside the
# instance can reach either of them.
cat > "$APP_DIR/docker-compose.ec2.yml" <<'COMPOSEEOF'
services:
  api:
    ports: !override
      - "80:8080"
  postgres:
    ports: !override
      - "127.0.0.1:5433:5432"
  redis:
    ports: !override
      - "127.0.0.1:6379:6379"
COMPOSEEOF

COMPOSE="docker compose -f docker-compose.yml -f docker-compose.ec2.yml"

# Build here rather than pulling from ECR: ECR's free tier is 500 MB and the
# LibreOffice image alone exceeds it. Building on the box is free, and only
# happens on deploy.
#
# One service at a time, deliberately. A plain `compose build` builds all
# four in parallel, and four concurrent Go compiles on 1 GB of RAM drives the
# load average past 17 and eats the entire swapfile - it thrashes for an hour
# instead of failing, which is worse than failing. Sequential is far faster
# here, and the images share most of their layers anyway.
#
# The API is built and started first so the site answers as early as
# possible; the workers (LibreOffice being much the largest) follow.
$COMPOSE build api
$COMPOSE up -d postgres redis

# Migrations run before the app, against the containerised Postgres.
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

# API first: the site is reachable from here on, while the workers build.
$COMPOSE up -d api

for svc in worker-pdf worker-image worker-office; do
  $COMPOSE build "$svc"
  $COMPOSE up -d "$svc"
done

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

echo "FileForge boot provisioning finished"
