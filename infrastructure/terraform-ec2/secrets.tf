# Secrets are generated here rather than passed in as variables.
#
# A secret supplied on the command line ends up in shell history, in CI logs
# and in any transcript of the session that set it. Generating them inside
# Terraform means the values exist in exactly two places: the state file and
# the instance's own .env - and neither is ever committed (.gitignore covers
# *.tfstate, and the .env is written on the box at boot).
#
# Read them back when you need them with:
#   terraform output -raw jwt_secret
resource "random_password" "jwt_secret" {
  length  = 64
  special = false # base62 keeps it safe to embed in env files and URLs unquoted
}

resource "random_password" "postgres" {
  length  = 32
  special = false # avoids quoting problems inside the Postgres connection string
}
