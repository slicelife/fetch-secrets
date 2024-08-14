
locals {
  role_name = "slice-service-${var.short_env}-${var.service_name}"
}

data "aws_ssm_parameter" "cluster_oidc_issuer_url" {
  name = "/devops/eks/cluster_oidc_issuer_url"
}

resource "aws_iam_policy" "service" {
  name = local.role_name
  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = [
          "secretsmanager:DescribeSecret",
          "secretsmanager:Get*"
        ],
        Resource = [
          "arn:aws:secretsmanager:*:*:secret:shared/*",
          "arn:aws:secretsmanager:*:*:secret:${var.service_name}/*"
        ]
      },
      {
        Effect = "Allow",
        Action = "iam:ListRoleTags",
        Resource = [
          "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${local.role_name}"
        ]
      },
      {
        Effect   = "Allow",
        Action   = "sts:GetCallerIdentity",
        Resource = ["*"]
      }
    ]
  })
}

module "aws_iam_role" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-assumable-role-with-oidc"
  version = "5.0.0"

  create_role                  = true
  role_name                    = local.role_name
  provider_url                 = nonsensitive(data.aws_ssm_parameter.cluster_oidc_issuer_url.value)
  role_policy_arns             = [aws_iam_policy.service.arn]
  oidc_subjects_with_wildcards = ["system:serviceaccount:${var.kube_namespace}*:*${var.service_name}"]
  tags = {
    secrets_default = "${var.service_name}/${var.short_env}/secrets"
  }
}

output "iam_role" {
  value = module.aws_iam_role.iam_role_arn
  depends_on = [
    module.aws_iam_role
  ]
}
