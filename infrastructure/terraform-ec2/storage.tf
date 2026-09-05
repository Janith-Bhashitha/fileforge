resource "aws_s3_bucket" "files" {
  # Bucket names are globally unique across all of AWS, so a fixed name
  # would collide with anyone else who ran this. The account ID suffix
  # makes it unique without needing manual input.
  bucket = "${local.name}-files-${data.aws_caller_identity.current.account_id}"
}

data "aws_caller_identity" "current" {}

# Free tier gives 5 GB of S3. Uploaded files are disposable by design, so
# expiring them keeps storage flat instead of creeping toward a bill.
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

# Needed for the presigned upload flow: the browser PUTs straight to S3
# from the app's origin, which the bucket has to permit.
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
