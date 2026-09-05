resource "aws_s3_bucket" "files" {
  bucket = "${local.name}-files"
}

# Uploaded files are transient by design — the app deletes them on its own
# retention schedule, and this lifecycle rule is the backstop for anything
# the cleanup task misses. Without it, a missed delete leaks storage cost
# forever.
resource "aws_s3_bucket_lifecycle_configuration" "files" {
  bucket = aws_s3_bucket.files.id

  rule {
    id     = "expire-files"
    status = "Enabled"

    filter {}

    expiration {
      days = var.file_retention_days
    }

    abort_incomplete_multipart_upload {
      days_after_initiation = 1
    }
  }
}

# Nothing in this bucket is ever meant to be publicly readable: clients reach
# objects through short-lived presigned URLs, never a public path.
resource "aws_s3_bucket_public_access_block" "files" {
  bucket = aws_s3_bucket.files.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "files" {
  bucket = aws_s3_bucket.files.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_versioning" "files" {
  bucket = aws_s3_bucket.files.id

  versioning_configuration {
    # Off deliberately: these objects are disposable, and versioning would
    # keep paying for every superseded conversion output.
    status = "Suspended"
  }
}

# CORS is required for the presigned upload flow — the browser PUTs straight
# to S3 from the app's own origin.
resource "aws_s3_bucket_cors_configuration" "files" {
  bucket = aws_s3_bucket.files.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["PUT", "GET", "HEAD"]
    allowed_origins = ["*"]
    expose_headers  = ["ETag"]
    max_age_seconds = 3000
  }
}

resource "aws_ecr_repository" "services" {
  for_each = toset(["api", "worker-pdf", "worker-image", "worker-office"])

  name                 = "${local.name}/${each.key}"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

# Untagged layers pile up on every deploy; ten tagged images is plenty of
# rollback history for a project this size.
resource "aws_ecr_lifecycle_policy" "services" {
  for_each   = aws_ecr_repository.services
  repository = each.value.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep the last 10 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 10
      }
      action = { type = "expire" }
    }]
  })
}
