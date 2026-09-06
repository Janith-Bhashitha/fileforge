# Deploying FileForge to EC2

A single t3.micro running the whole docker-compose stack, plus an S3 bucket
for file storage. The box never builds an image — it pulls the six that CI
already pushed to Docker Hub.

## Order matters

The instance pulls `DOCKERHUB_USERNAME/fileforge-*:latest`. Those images come
from CI, which builds on push to `main`. **Deploying before CI has built your
latest commit ships the previous code**, silently — the deploy succeeds and
serves the old build.

```
commit → push → CI builds & pushes 6 images → terraform apply
```

Check the Actions tab is green before applying.

## First-time setup

```bash
export AWS_PROFILE=your-profile          # or AWS_ACCESS_KEY_ID / SECRET
export TF_VAR_dockerhub_username=janithbhashitha
export TF_VAR_gemini_api_key=...         # optional; AI Processing needs it
export TF_VAR_smtp_username=you@gmail.com    # optional; password-reset email
export TF_VAR_smtp_password=...              # Gmail App Password, not your password

terraform init
terraform apply
```

`terraform output app_url` gives the address. Allow a few minutes on first
boot: the instance pulls six images and runs migrations before the site
answers.

## Changing `user_data.sh` replaces the instance

`ec2.tf` sets `user_data_replace_on_change = true`, because user_data only
ever executes on first boot — an edit that isn't accompanied by a rebuild
would be silently ignored.

**The consequence is that editing `user_data.sh` destroys the database.**
Postgres runs in a container on a docker volume on the instance's root disk,
so replacing the instance takes the volume with it: accounts, jobs, batches
and file records all go. Objects in S3 survive, but the rows describing them
don't, so they become orphaned until the lifecycle rule expires them.

Run `terraform plan` and look for `must be replaced` before applying. If you
need the data, dump it first over SSM:

```bash
aws ssm start-session --target "$(terraform output -raw instance_id)"
cd /opt/fileforge
docker compose -f docker-compose.yml -f docker-compose.ec2.yml \
  exec -T postgres pg_dump -U fileforge fileforge > /tmp/backup.sql
```

## Deploying new code without replacing the instance

For an ordinary code change — no `user_data.sh` edit — nothing needs
Terraform at all. The nightly `fileforge-redeploy` cron pulls the newest
images, applies migrations and restarts what changed. To do it immediately:

```bash
aws ssm start-session --target "$(terraform output -raw instance_id)"
sudo /etc/cron.daily/fileforge-redeploy
```

That script runs `migrate up` before starting the new containers. Migrations
are **not** optional on this path: user_data's migration step only ever runs
on first boot, so without them a redeploy starts new code against the schema
the box was born with, and every request touching a new table fails.

## Troubleshooting

Provisioning output, and the first place to look if the site doesn't come up:

```bash
aws ssm start-session --target "$(terraform output -raw instance_id)"
sudo tail -100 /var/log/cloud-init-output.log
```

Port 22 is closed by default (`ssh_ingress_cidr` defaults to a loopback CIDR
matching nobody). SSM Session Manager needs no open port and is the intended
way in. Set `ssh_ingress_cidr` to `YOUR.IP/32` only if you specifically want
SSH — never `0.0.0.0/0`.

## Local development

The stack runs locally with `docker compose up -d`, but migrations are not
applied automatically there either. With Postgres up:

```bash
migrate -path services/api/migrations \
  -database "postgres://fileforge:fileforge@localhost:5433/fileforge?sslmode=disable" up
```

`migrate` is [golang-migrate](https://github.com/golang-migrate/migrate).
Without this the API starts but every query against a missing table fails.

Note that `docker compose build` produces several GB of build cache. If disk
is tight, `docker builder prune -af` reclaims it (build cache only — it does
not touch volumes, so the database is safe).
