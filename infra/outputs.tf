output "service_url" {
  description = "Public HTTPS URL of the App Runner service."
  value       = "https://${aws_apprunner_service.app.service_url}"
}

output "ecr_repository_url" {
  description = "Pass as AWS_ACCOUNT_ID-derived registry to `make docker-push`."
  value       = aws_ecr_repository.app.repository_url
}

output "apprunner_service_arn" {
  description = "Set as the APPRUNNER_SERVICE_ARN repo variable for CI."
  value       = aws_apprunner_service.app.arn
}

output "github_ci_role_arn" {
  description = "Set as the AWS_ROLE_ARN repo variable for CI (null if github_repo unset)."
  value       = var.github_repo == "" ? null : aws_iam_role.github_ci[0].arn
}
