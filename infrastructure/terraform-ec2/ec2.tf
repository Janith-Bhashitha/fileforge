resource "aws_security_group" "app" {
  name        = "${local.name}-app"
  description = "FileForge single-instance deployment"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "Web UI and API (HTTP)"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Modern browsers - Chrome's HTTPS-First Mode especially, on by default and
  # more aggressive on mobile - rewrite a bare address typed with no scheme
  # to https://. Without this rule those connections don't fail fast, they
  # hang for the full timeout: identical, from a user's perspective, to the
  # server being down. nginx answers here with a self-signed certificate
  # (see infrastructure/docker/web) since there's no domain name to get a
  # real one validated against.
  ingress {
    description = "Web UI and API (HTTPS, self-signed)"
    from_port   = 443
    to_port     = 443
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
    repo_url           = var.repo_url
    repo_branch        = var.repo_branch
    s3_bucket          = aws_s3_bucket.files.id
    region             = var.region
    jwt_secret         = random_password.jwt_secret.result
    postgres_password  = random_password.postgres.result
    dockerhub_username = var.dockerhub_username
    gemini_api_key     = var.gemini_api_key
    gemini_model       = var.gemini_model

    # http, not https: the box answers 443 with a self-signed certificate, so
    # an emailed https link lands the recipient on a browser interstitial.
    # A full URL with an explicit scheme isn't subject to the HTTPS-First
    # upgrade that bare typed addresses are, so this stays http end to end.
    frontend_url  = "http://${aws_eip.app.public_ip}"
    smtp_host     = var.smtp_host
    smtp_port     = var.smtp_port
    smtp_username = var.smtp_username
    smtp_password = var.smtp_password
    smtp_from     = var.smtp_from
  })

  # Changing user_data on an existing instance does nothing (it only runs on
  # first boot), so treat a change to it as "rebuild the box".
  user_data_replace_on_change = true

  tags = { Name = local.name }
}

# A static address that survives stop/start. Free while attached to a
# running instance - but AWS charges for an Elastic IP that is allocated
# and NOT attached, which is the classic surprise line item.
#
# The address is allocated on its own and attached separately, rather than
# with the `instance` argument, because user_data needs to bake the public
# address into FRONTEND_URL. Associating here would make the EIP depend on
# the instance while the instance depends on the EIP's address - a cycle
# Terraform refuses to plan. Allocating first breaks it: the address exists
# before the instance that gets told about it.
resource "aws_eip" "app" {
  domain = "vpc"

  tags = { Name = local.name }
}

resource "aws_eip_association" "app" {
  instance_id   = aws_instance.app.id
  allocation_id = aws_eip.app.id
}
