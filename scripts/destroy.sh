#!/usr/bin/env bash
#
# Tear down everything Terraform created (App Runner service, ECR repo, IAM
# roles). Does NOT delete the S3 state bucket — that was created out-of-band by
# create-s3.sh; remove it manually if you truly want a clean slate.
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

read -rp "This destroys the App Runner service, ECR repo, and IAM roles. Type 'yes' to continue: " ans
[ "$ans" = "yes" ] || { echo "Aborted."; exit 1; }

cd "$ROOT/infra"
tofu destroy
