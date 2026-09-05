resource "aws_db_subnet_group" "main" {
  name       = local.name
  subnet_ids = aws_subnet.private[*].id
}

resource "aws_db_instance" "main" {
  identifier     = local.name
  engine         = "postgres"
  engine_version = "16"
  instance_class = var.db_instance_class

  allocated_storage     = 20
  max_allocated_storage = 100
  storage_type          = "gp3"
  storage_encrypted     = true

  db_name  = "fileforge"
  username = var.db_username
  password = var.db_password

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.database.id]
  publicly_accessible    = false

  # Single-AZ and a short backup window: cost discipline for a portfolio
  # deployment. Both are one-line changes if this ever needs to be real.
  multi_az                = false
  backup_retention_period = 1
  skip_final_snapshot     = true

  auto_minor_version_upgrade = true

  tags = { Name = local.name }
}

resource "aws_elasticache_subnet_group" "main" {
  name       = local.name
  subnet_ids = aws_subnet.private[*].id
}

resource "aws_security_group" "redis" {
  name   = "${local.name}-redis"
  vpc_id = aws_vpc.main.id

  ingress {
    description     = "Redis from application tasks"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.tasks.id]
  }

  tags = { Name = "${local.name}-redis" }
}

# The queue is Redis Streams, which ElastiCache serves unchanged — the
# application connection string is the only thing that differs from the
# local docker-compose Redis.
resource "aws_elasticache_cluster" "main" {
  cluster_id           = local.name
  engine               = "redis"
  engine_version       = "7.1"
  node_type            = "cache.t4g.micro"
  num_cache_nodes      = 1
  parameter_group_name = "default.redis7"
  port                 = 6379

  subnet_group_name  = aws_elasticache_subnet_group.main.name
  security_group_ids = [aws_security_group.redis.id]
}

# Secrets live in Secrets Manager and are injected into tasks by ARN, so a
# task definition never carries the value itself.
resource "aws_secretsmanager_secret" "app" {
  name                    = "${local.name}-app"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "app" {
  secret_id = aws_secretsmanager_secret.app.id

  secret_string = jsonencode({
    DATABASE_URL = "postgres://${var.db_username}:${var.db_password}@${aws_db_instance.main.endpoint}/fileforge?sslmode=require"
    REDIS_URL    = "redis://${aws_elasticache_cluster.main.cache_nodes[0].address}:6379"
    JWT_SECRET   = var.jwt_secret
  })
}
