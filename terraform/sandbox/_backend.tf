terraform {
  backend "s3" {
    bucket   = "202758899153-terraform-state"
    key      = "fetch-secrets/sbx/terraform.tfstate"
    region   = "us-east-2"
    role_arn = "arn:aws:iam::202758899153:role/terraform"
    encrypt  = true
  }
}


