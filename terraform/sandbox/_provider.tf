variable "aws_role_arn" {
  description = "Optional role arn for terraform to assume"
  default     = "arn:aws:iam::202758899153:role/terraform"
}

provider "aws" {
  region = "us-east-2"

  assume_role {
    role_arn = var.aws_role_arn
  }

  default_tags {
    tags = {
      terraform   = "true"
      git         = "https://github.com/slicelife/fetch-secrets"
      environment = "sandbox"
      department  = "devops"
      subteam     = "devops"
    }
  }
}
