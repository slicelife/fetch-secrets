# Fetch secrets from AWS Secrets Manager

---

## Usage

This small binary fetches JSON formatted secrets from AWS Secrets Manager.
It exports the fetched secrets to the shell environment in which some binary (that requires said secrets) is then executed.

The secrets should be stored in a AWS Secrets Manager secret with a value specified in a flat JSON K:V format.
The Secrets Manager secret name should usually follow the path pattern `$service_name/$env/secrets`.  


`fetch-secrets` will use the IAM role tags to auto-discover the secrets to be fetched.
Any tag keys prefixed with `secrets_` will be matched; the corresponding tag value be a Secrets Manager secret name.  
eg; The tag `secrets_default = myservice/dev/secrets` will load the secrets stored in the secret named `myservice/dev/secrets`. 
You can specify multiple tags, however, the merging order is arbitrary so don't expect precedence.

---

## Running

```shell
./fetch-secrets mycommand subcommand --my-arg example
```
This will load the variables and exec `mycommand` (with the subcommand and args), making the secret values available as env vars for the `mycommand` application.

###  IAM policy required for `fetch-secrets`

In order to auto-discover secrets, `fetch-secrets` or the container/pod/instance it is running on requires the following IAM policy:  
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

---