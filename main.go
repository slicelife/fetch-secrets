package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sts"
)

func fetch_secrets_from_path(secretId string, sess *session.Session) interface{} {
	secret_manager_svc := secretsmanager.New(sess)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretId),
	}

	result, err := secret_manager_svc.GetSecretValue(input)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			log.Fatal(err.Error())
		}
	}

	var secretString string
	secretString = *result.SecretString

	var anyJson interface{}

	json.Unmarshal([]byte(secretString), &anyJson)

	return anyJson
}

func flatten_json(secretsMap interface{}) []string {
	var tmp []string
	for key, el := range secretsMap.(map[string]interface{}) {
		tmp = append(tmp, fmt.Sprintf("%s=%s", key, el))
	}

	return tmp
}

func get_role_name(sess *session.Session) string {
	sts_svc := sts.New(sess)

	caller_identity, err := sts_svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatal(err.Error())
	}

	var role_arn string = *caller_identity.Arn

	split_role := strings.Split(role_arn, "/")

	var role_name string = split_role[1]

	return role_name
}

func main() {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err.Error())
	}
	iam_svc := iam.New(sess)

	var role_name = get_role_name(sess)

	tags_resp, err := iam_svc.ListRoleTags(&iam.ListRoleTagsInput{
		RoleName: &role_name,
	})

	var newEnv []string = os.Environ()

	for _, v := range tags_resp.Tags {
		if strings.HasPrefix(*v.Key, "secrets_") {
			fmt.Println(*v.Value)
			var fetched_secrets interface{} = fetch_secrets_from_path(*v.Value, sess)
			newEnv = append(newEnv, flatten_json(fetched_secrets)...)
		}
	}
	path, err := exec.LookPath(os.Args[1])

	if err := syscall.Exec(path, os.Args[1:], newEnv); err != nil {
		log.Fatal(err)
	}
}
