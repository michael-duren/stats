#!/usr/bin/env bash
#
# Redeploy: build + push a new image, then trigger an App Runner deployment.
# Run this for manual releases after the initial bootstrap. (CI does the same on
# push to main via .github/workflows/deploy.yml.)
#
# Env overrides: AWS_REGION (default us-east-1), APP (default ghstats).
set -euo pipefail

REGION="${AWS_REGION:-us-east-1}"
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

ACCT=$(aws sts get-caller-identity --query Account --output text)
make -C "$ROOT" docker-push AWS_ACCOUNT_ID="$ACCT" AWS_REGION="$REGION"

SERVICE_ARN=$(cd "$ROOT/infra" && tofu output -raw apprunner_service_arn)
echo "==> Triggering App Runner deployment"
aws apprunner start-deployment --service-arn "$SERVICE_ARN" --region "$REGION"
echo "Started. Track status in the App Runner console or with:"
echo "  aws apprunner describe-service --service-arn $SERVICE_ARN --region $REGION --query 'Service.Status'"
