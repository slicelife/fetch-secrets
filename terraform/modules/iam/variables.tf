data "aws_caller_identity" "current" {}

variable "service_name" {
  type    = string
  default = "core_api"
}

variable "short_env" {
  type = string
}

variable "kube_namespace" {
  type = string
}
