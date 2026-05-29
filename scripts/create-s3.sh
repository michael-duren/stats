#!/usr/bin env

# The state bucket has to exist before tofu init
#  can use it as a backend — Terraform can't store
#  state in a bucket it hasn't created yet. So you
#  bootstrap it with the AWS CLI, not Terraform:

ACCT=$(aws sts get-caller-identity --query Account --output text)

BUCKET="ghstats-tfstate-$ACCT" # S3 names are globally unique account id keeps it so

# 1. create (us-east-1 only — see note below)
aws s3api create-bucket --bucket "$BUCKET" --region us-east-1

# 2. versioning — lets you recover/roll back acorrupted state file
aws s3api put-bucket-versioning --bucket "$BUCKET" \
    --versioning-configuration Status=Enabled

# 3. block all public access (state contains
# your GITHUB_TOKEN)
aws s3api put-public-access-block --bucket "$BUCKET" \
    --public-access-block-configuration \
    BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true

# 4. encrypt at rest
aws s3api put-bucket-encryption --bucket "$BUCKET" \
    --server-side-encryption-configuration \
    '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'
