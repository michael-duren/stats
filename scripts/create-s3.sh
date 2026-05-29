#!/usr/bin/env bash
#
# Bootstrap the Terraform/OpenTofu remote-state backend.
#
# The state bucket must exist BEFORE `tofu init` can use it as a backend —
# Terraform can't store state in a bucket it hasn't created yet. So we create it
# with the AWS CLI, then write infra/backend.hcl for `tofu init -backend-config`.
#
# Env overrides: AWS_REGION (default us-east-1), APP (default ghstats).
set -euo pipefail

REGION="${AWS_REGION:-us-east-1}"
APP="${APP:-ghstats}"

ACCT=$(aws sts get-caller-identity --query Account --output text)
BUCKET="${APP}-tfstate-${ACCT}" # S3 names are globally unique; account id keeps it so

echo "Account: $ACCT"
echo "Region:  $REGION"
echo "Bucket:  $BUCKET"

# 1. create the bucket (idempotent). us-east-1 must NOT send a LocationConstraint;
#    every other region must.
if aws s3api head-bucket --bucket "$BUCKET" 2>/dev/null; then
  echo "Bucket already exists, skipping create."
elif [ "$REGION" = "us-east-1" ]; then
  aws s3api create-bucket --bucket "$BUCKET" --region "$REGION"
else
  aws s3api create-bucket --bucket "$BUCKET" --region "$REGION" \
    --create-bucket-configuration "LocationConstraint=$REGION"
fi

# 2. versioning — lets you recover/roll back a corrupted state file
aws s3api put-bucket-versioning --bucket "$BUCKET" \
  --versioning-configuration Status=Enabled

# 3. block all public access (state can contain secrets like GITHUB_TOKEN)
aws s3api put-public-access-block --bucket "$BUCKET" \
  --public-access-block-configuration \
  BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true

# 4. encrypt at rest
aws s3api put-bucket-encryption --bucket "$BUCKET" \
  --server-side-encryption-configuration \
  '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'

# 5. emit the backend config consumed by `tofu init -backend-config=backend.hcl`.
#    use_lockfile gives S3-native state locking (no DynamoDB table needed) on
#    OpenTofu >= 1.10 / Terraform >= 1.11.
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
BACKEND="$SCRIPT_DIR/../infra/backend.hcl"
cat >"$BACKEND" <<EOF
bucket       = "$BUCKET"
key          = "$APP/terraform.tfstate"
region       = "$REGION"
encrypt      = true
use_lockfile = true
EOF

echo "Wrote $BACKEND"
echo "Next: cd infra && tofu init -backend-config=backend.hcl"
