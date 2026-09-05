resource "aws_security_group" "app" {
  name        = "${local.name}-app"
  description = "FileForge single-instance deployment"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "Web UI and API"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # SSH is closed by default (the variable defaults to a loopback CIDR that
  # matches nobody). Port 22 open to 0.0.0.0/0 on a free-tier box is one of
  # the most reliably exploited things on AWS.
  ingress {
    description = "SSH, only from the configured address"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.ssh_ingress_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${local.name}-app" }
}

# The instance reaches S3 through this role rather than credentials baked
# into the AMI or the compose file. Nothing secret ever lands on the disk.
resource "aws_iam_role" "instance" {
  name = "${local.name}-instance"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

# Scoped to this one bucket, not s3:* across the account.
resource "aws_iam_role_policy" "instance_s3" {
  name = "files-bucket-access"
  role = aws_iam_role.instance.id

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

# SSM Session Manager gives shell access through the AWS console with no
# open port and no SSH key - strictly better than exposing 22.
resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.instance.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "instance" {
  name = "${local.name}-instance"
  role = aws_iam_role.instance.name
}

resource "aws_instance" "app" {
  ami                    = data.aws_ssm_parameter.al2023.value
  instance_type          = var.instance_type
  subnet_id              = data.aws_subnets.default.ids[0]
  vpc_security_group_ids = [aws_security_group.app.id]
  iam_instance_profile   = aws_iam_instance_profile.instance.name
  key_name               = var.key_pair_name != "" ? var.key_pair_name : null

  root_block_device {
    volume_size = var.root_volume_gb
    volume_type = "gp3"
    encrypted   = true
  }

  user_data = templatefile("${path.module}/user_data.sh", {
    repo_url          = var.repo_url
    repo_branch       = var.repo_branch
    s3_bucket         = aws_s3_bucket.files.id
    region            = var.region
    jwt_secret        = random_password.jwt_secret.result
    postgres_password = random_password.postgres.result
  })

  # Changing user_data on an existing instance does nothing (it only runs on
  # first boot), so treat a change to it as "rebuild the box".
  user_data_replace_on_change = true

  tags = { Name = local.name }
}

# A static address that survives stop/start. Free while attached to a
# running instance - but AWS charges for an Elastic IP that is allocated
# and NOT attached, which is the classic surprise line item.
resource "aws_eip" "app" {
  instance = aws_instance.app.id
  domain   = "vpc"

  tags = { Name = local.name }
}
