resource "aws_ecs_cluster" "main" {
  name = local.name

  setting {
    name  = "containerInsights"
    value = "disabled" # costs money; CloudWatch logs plus /metrics is enough here
  }
}

resource "aws_cloudwatch_log_group" "services" {
  for_each = toset(concat(["api"], keys(local.workers)))

  name              = "/ecs/${local.name}/${each.key}"
  retention_in_days = 7
}

# The execution role is what ECS itself uses to pull images and read secrets;
# the task role is what the running container gets. Keeping them separate is
# what stops application code from inheriting the ability to pull images or
# read every secret.
resource "aws_iam_role" "execution" {
  name = "${local.name}-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "execution" {
  role       = aws_iam_role.execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy" "execution_secrets" {
  name = "read-app-secrets"
  role = aws_iam_role.execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["secretsmanager:GetSecretValue"]
      Resource = [aws_secretsmanager_secret.app.arn]
    }]
  })
}

resource "aws_iam_role" "task" {
  name = "${local.name}-task"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

# Scoped to this bucket's objects only — not s3:* on every bucket in the
# account, which is the easy mistake here.
resource "aws_iam_role_policy" "task_s3" {
  name = "files-bucket-access"
  role = aws_iam_role.task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
        Resource = ["${aws_s3_bucket.files.arn}/*"]
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = [aws_s3_bucket.files.arn]
      }
    ]
  })
}

locals {
  common_env = [
    { name = "STORAGE_BACKEND", value = "s3" },
    { name = "S3_BUCKET", value = aws_s3_bucket.files.id },
    { name = "S3_REGION", value = var.region },
    { name = "STORAGE_DIR", value = "/data/storage" },
    { name = "API_PORT", value = "8080" },
  ]

  common_secrets = [
    { name = "DATABASE_URL", valueFrom = "${aws_secretsmanager_secret.app.arn}:DATABASE_URL::" },
    { name = "REDIS_URL", valueFrom = "${aws_secretsmanager_secret.app.arn}:REDIS_URL::" },
    { name = "JWT_SECRET", valueFrom = "${aws_secretsmanager_secret.app.arn}:JWT_SECRET::" },
  ]
}

resource "aws_ecs_task_definition" "api" {
  family                   = "${local.name}-api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 512
  memory                   = 1024
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([{
    name      = "api"
    image     = var.api_image != "" ? var.api_image : "${aws_ecr_repository.services["api"].repository_url}:latest"
    essential = true

    portMappings = [{ containerPort = 8080, protocol = "tcp" }]
    environment  = local.common_env
    secrets      = local.common_secrets

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.services["api"].name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "api"
      }
    }

    healthCheck = {
      command     = ["CMD-SHELL", "wget -q -O- http://localhost:8080/healthz || exit 1"]
      interval    = 30
      timeout     = 5
      retries     = 3
      startPeriod = 30
    }
  }])
}

# Each worker gets its own task definition and service, which is the payoff
# of having built them as separate containers back in Phase 3: the office
# worker can scale (or be sized) independently of the PDF one.
resource "aws_ecs_task_definition" "workers" {
  for_each = local.workers

  family                   = "${local.name}-${each.key}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = each.value.cpu
  memory                   = each.value.memory
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([{
    name      = each.key
    image     = "${aws_ecr_repository.services[each.key].repository_url}:latest"
    essential = true

    environment = local.common_env
    secrets     = local.common_secrets

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.services[each.key].name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = each.key
      }
    }
  }])
}

resource "aws_lb" "main" {
  name               = local.name
  load_balancer_type = "application"
  subnets            = aws_subnet.public[*].id
  security_groups    = [aws_security_group.alb.id]
}

resource "aws_lb_target_group" "api" {
  name        = "${local.name}-api"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = aws_vpc.main.id
  target_type = "ip"

  health_check {
    path                = "/healthz"
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 30
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.main.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api.arn
  }
}

resource "aws_ecs_service" "api" {
  name            = "${local.name}-api"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.api.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.public[*].id
    security_groups  = [aws_security_group.tasks.id]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.api.arn
    container_name   = "api"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.http]

  # CI updates the task definition on deploy; Terraform shouldn't fight it
  # by reverting to whatever revision was current at the last apply.
  lifecycle {
    ignore_changes = [task_definition]
  }
}

resource "aws_ecs_service" "workers" {
  for_each = local.workers

  name            = "${local.name}-${each.key}"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.workers[each.key].arn
  desired_count   = each.value.count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.public[*].id
    security_groups  = [aws_security_group.tasks.id]
    assign_public_ip = true
  }

  lifecycle {
    ignore_changes = [task_definition]
  }
}
