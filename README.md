# Fetch secrets from AWS Secrets Manager

---

## Usage

This small binary fetches JSON formatted secrets from AWS Secrets Manager.
It passes the fetched secrets to the binary (that requires the secrets) when it is executed. (see [`execve`](https://man7.org/linux/man-pages/man2/execve.2.html))

The secrets should be stored in a AWS Secrets Manager secret with a value specified in a flat JSON K:V format.
The Secrets Manager secret name should usually follow the path pattern `$service_name/$env/secrets`.  


`fetch-secrets` will use the IAM role tags to auto-discover the secrets to be fetched.
Any tag keys prefixed with `secrets_` will be matched; the corresponding tag value be a Secrets Manager secret name.  
eg; The tag `secrets_default = myservice/dev/secrets` will load the secrets stored in the secret named `myservice/dev/secrets`. 
You can specify multiple tags, however, the merging order is arbitrary so don't expect precedence.

`fetch-secrets` supports the use of an `FS_REGION` env var, which can be configured when applications use different regions
for their secrets.  Note, the application will still use `AWS_REGION` as normal.

---

## Running

```shell
./fetch-secrets mycommand subcommand --my-arg example
```
This will load the variables and exec `mycommand` (with the subcommand and args), making the secret values available as env vars for the `mycommand` application.

To run `fetch-secrets` in a docker container, download the [latest binary release](https://github.com/slicelife/fetch-secrets/releases), update the binary permissions and prepend to your existing app command. eg;
```dockerfile
ADD https://github.com/slicelife/fetch-secrets/releases/download/v0.2.0/fetch-secrets-v0.2.0-linux-amd64 /fetch-secrets
RUN chmod +x /fetch-secrets
CMD ["/fetch-secrets", "/entrypoint.sh"]
```

###  IAM policy required for `fetch-secrets`

In order to auto-discover secrets, `fetch-secrets` or the container/pod/instance it is running on requires the following IAM policy:
```hcl
resource "aws_iam_policy" "service" {
  name   = local.role_name
  policy = data.aws_iam_policy_document.service.json
}

data "aws_iam_policy_document" "service" {
  statement {
    actions = ["secretsmanager:GetSecretValue"]
    resources = [
      "arn:aws:secretsmanager:*:${data.aws_caller_identity.current.account_id}:secret:${var.service_name}/${var.short_env}/*"
    ]
    effect = "Allow"
  }

  statement {
    actions   = ["iam:ListRoleTags"]
    resources = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${local.role_name}"]
    effect    = "Allow"
  }

  statement {
    actions   = ["sts:GetCallerIdentity"]
    resources = ["*"]
    effect    = "Allow"
  }
}
```
Or alternatively using embedded JSON:
```terraform
policy = jsonencode({
    "Version" : "2012-10-17",
    "Statement" : [
      {
        "Effect" : "Allow",
        "Action" : "secretsmanager:GetSecretValue",
        "Resource" : [
          "arn:aws:secretsmanager:*:${data.aws_caller_identity.current.account_id}:secret:${var.service_name}/${var.short_env}/*"
        ]
      },
      {
        "Effect" : "Allow",
        "Action" : "iam:ListRoleTags",
        "Resource" : [
          "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${local.role_name}"
        ]
      },
      {
        "Effect" : "Allow",
        "Action" : "sts:GetCallerIdentity",
        "Resource" : ["*"]
      }
    ]
  })
```

You can add tags to your role using Terraform.  eg;
```terraform
  tags = {
    secrets_default = "${var.service_name}/${var.short_env}/secrets"
  }
```

---

## Building, testing and linting

Build the binary using:  
```shell
make artifact
```
The resulting binary will be called `fetch-secrets`.

_NOTE: If you need to compile a linux specific binary run and you're on a Mac use:_
```shell
GOOS=linux go build
```

Test the code using:
```shell
make test
```

Lint the code using:
```shell
make lint
```

Generate mocks for unit testing:
```shell
make generate
```

---

## Troubleshooting

`fetch-secrets` uses JSON logging throughout.  The logging should indicate, the role determined, the tag keys and values as well as the secret-names to fetch values for.  
If logs are unavailable, the following exit-codes should be helpful in indicating where the issue might be:

| Exit Code | Description                             |
|:---------:|-----------------------------------------|
|     1     | Executable to run not found             |
|     2     | Problem loading AWS config              |
|     3     | Failed to get AWS role, tags or secrets |
|     4     | Timeout (default 1min)                  |
|     5     | Failed to run executable                |

---
