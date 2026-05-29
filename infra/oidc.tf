# GitHub Actions OIDC role: lets .github/workflows/deploy.yml push to ECR and
# trigger App Runner deployments without static AWS keys. Enabled only when
# var.github_repo is set.
#
# AWS only permits one OIDC provider per URL per account, so we *reference* the
# existing token.actions.githubusercontent.com provider as a data source rather
# than trying to create a duplicate. If your account has never used GitHub
# Actions before, create it once with:
#   aws iam create-open-id-connect-provider \
#     --url https://token.actions.githubusercontent.com \
#     --client-id-list sts.amazonaws.com \
#     --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1

data "aws_iam_openid_connect_provider" "github" {
  count = var.github_repo == "" ? 0 : 1
  url   = "https://token.actions.githubusercontent.com"
}

data "aws_iam_policy_document" "github_assume" {
  count = var.github_repo == "" ? 0 : 1

  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.github[0].arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${var.github_repo}:*"]
    }
  }
}

resource "aws_iam_role" "github_ci" {
  count              = var.github_repo == "" ? 0 : 1
  name               = "${var.app_name}-github-ci"
  assume_role_policy = data.aws_iam_policy_document.github_assume[0].json
}

data "aws_iam_policy_document" "github_ci" {
  count = var.github_repo == "" ? 0 : 1

  statement {
    sid       = "ECRAuth"
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }

  statement {
    sid = "ECRPush"
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:BatchGetImage",
      "ecr:CompleteLayerUpload",
      "ecr:GetDownloadUrlForLayer",
      "ecr:InitiateLayerUpload",
      "ecr:PutImage",
      "ecr:UploadLayerPart",
    ]
    resources = [aws_ecr_repository.app.arn]
  }

  statement {
    sid       = "AppRunnerDeploy"
    actions   = ["apprunner:StartDeployment"]
    resources = [aws_apprunner_service.app.arn]
  }
}

resource "aws_iam_role_policy" "github_ci" {
  count  = var.github_repo == "" ? 0 : 1
  name   = "${var.app_name}-github-ci"
  role   = aws_iam_role.github_ci[0].id
  policy = data.aws_iam_policy_document.github_ci[0].json
}
