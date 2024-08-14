// development
module "service_iam_role" {
  source = "../modules/iam"

  short_env    = "sbx"
  service_name = "fetch-secrets"

  kube_namespace = "devops"
}


output "iam_role" {
  value = module.service_iam_role.iam_role
}
