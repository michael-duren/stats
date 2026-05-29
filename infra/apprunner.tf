# Role App Runner assumes to pull the image from your private ECR repo.
data "aws_iam_policy_document" "apprunner_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["build.apprunner.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "apprunner_ecr" {
  name               = "${var.app_name}-apprunner-ecr"
  assume_role_policy = data.aws_iam_policy_document.apprunner_assume.json
}

resource "aws_iam_role_policy_attachment" "apprunner_ecr" {
  role       = aws_iam_role.apprunner_ecr.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSAppRunnerServicePolicyForECRAccess"
}

resource "aws_apprunner_service" "app" {
  service_name = var.app_name

  source_configuration {
    # CI calls apprunner:StartDeployment after pushing; no auto-deploy on push.
    auto_deployments_enabled = false

    authentication_configuration {
      access_role_arn = aws_iam_role.apprunner_ecr.arn
    }

    image_repository {
      image_identifier      = "${aws_ecr_repository.app.repository_url}:${var.image_tag}"
      image_repository_type = "ECR"

      image_configuration {
        port = "8080"

        # NOTE: this lands in Terraform state. For stronger secrecy, store the
        # token in SSM Parameter Store / Secrets Manager and use
        # runtime_environment_secrets instead.
        runtime_environment_variables = {
          GITHUB_TOKEN     = var.github_token
          ALLOWED_USERNAME = var.allowed_username
        }
      }
    }
  }

  health_check_configuration {
    protocol = "HTTP"
    path     = "/healthz"
  }

  instance_configuration {
    cpu    = var.cpu
    memory = var.memory
  }
}
