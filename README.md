### Fetch secrets from AWS Secrets Manager

#### Usage

This small binary fetches JSON formatted secrets from AWS Secrets Manager.  
The secrets should be in a flat JSON K:V format. 
The secrets path should usually follow the pattern `$service_name/$env/secrets`  


It will use the the IAM role tags to autodiscover the secrets to be fetched, 
any tags begginig with `secrets_` will be matched, the value should point to the secrets manager path.  
For example the tag `secrets_default = myservice/dev/secrets` will load the secrets at that location. You can specify multiple tags, the merging order is arbitrary so don't expect precedence.


You then run it like this:  
```sh
./fetch-secrets mycommand subcommand --my-arg example
```
This will load the variables and exec `mycommand` making them available as env variables in your application.

#### IAM configuration
It will need the following policy to auto discover secrets:  
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

You can then add tags to your role:  
```terraform
  tags = {
    secrets_default = "${var.service_name}/${var.short_env}/secrets"
  }
```


#### Building

Run the following commands:  
```sh
go get
go build
````

The resulting binary will be in this directory called `fetch-secrets`.

If you need to compile a linux specific binary run and you're on a mac run:  
`GOOS=linux go build`
