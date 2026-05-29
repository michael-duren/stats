#!/usr/bin/env bash
#
# First-time deploy. Resolves the chicken-and-egg where App Runner needs an image
# to already exist before the service can be created:
#   1. init the S3 backend
#   2. create the ECR repo
#   3. build + push the first image
#   4. create the App Runner service + IAM roles
#
# Prereqs: scripts/create-s3.sh has been run, infra/terraform.tfvars exists, and
#   export TF_VAR_github_token=ghp_yourtoken
#
# Env overrides: AWS_REGION (default us-east-1), APP (default ghstats).
set -euo pipefail

REGION="${AWS_REGION:-us-east-1}"
APP="${APP:-ghstats}"
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

: "${TF_VAR_github_token:?Set TF_VAR_github_token=ghp_... before running}"

cd "$ROOT/infra"
[ -f backend.hcl ] || {
  echo "infra/backend.hcl missing — run scripts/create-s3.sh first." >&2
  exit 1
}
[ -f terraform.tfvars ] || echo "warning: infra/terraform.tfvars not found; using variable defaults (ALLOWED_USERNAME will be empty)."

tofu init -backend-config=backend.hcl

echo "==> 1/3 Creating ECR repository"
tofu apply -auto-approve -target=aws_ecr_repository.app

echo "==> 2/3 Building and pushing the first image"
ACCT=$(aws sts get-caller-identity --query Account --output text)
make -C "$ROOT" docker-push AWS_ACCOUNT_ID="$ACCT" AWS_REGION="$REGION"

echo "==> 3/3 Applying the rest of the infrastructure"
tofu apply -auto-approve

echo
echo "Service URL: $(tofu output -raw service_url)"
