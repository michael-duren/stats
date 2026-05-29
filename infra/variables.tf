variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "app_name" {
  type    = string
  default = "ghstats"
}

variable "image_tag" {
  type        = string
  default     = "latest"
  description = "ECR image tag App Runner runs."
}

variable "github_token" {
  type        = string
  sensitive   = true
  description = "GitHub PAT (no scopes needed for public data). Pass via env: export TF_VAR_github_token=ghp_xxx."
}

variable "cpu" {
  type        = string
  default     = "0.25 vCPU"
  description = "App Runner instance CPU."
}

variable "memory" {
  type        = string
  default     = "0.5 GB"
  description = "App Runner instance memory."
}

variable "allowed_username" {
  type        = string
  default     = ""
  description = "Locks the instance to a single GitHub username (sets ALLOWED_USERNAME). Empty string serves any username."
}

variable "github_repo" {
  type        = string
  default     = "michael-duren/stats"
  description = "owner/name of the GitHub repo allowed to assume the CI deploy role. Empty string disables the OIDC role entirely."
}
