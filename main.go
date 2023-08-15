package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func fetch_secrets_from_path(secretId string, cfg aws.Config) interface{} {
	secret_manager_svc := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretId),
	}

	result, err := secret_manager_svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		log.Fatal(err)
	}

	var secretString string = *result.SecretString

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

func get_role_name(cfg aws.Config) string {
	sts_svc := sts.NewFromConfig(cfg)

	caller_identity, err := sts_svc.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatal(err.Error())
	}

	var role_arn string = *caller_identity.Arn

	split_role := strings.Split(role_arn, "/")

	var role_name string = split_role[1]

	return role_name
}

func main() {
	var region_config config.LoadOptionsFunc

	manual_region, exists := os.LookupEnv("FS_REGION")
	if exists {
		region_config = config.WithRegion(manual_region)
	} else {
		region_config = config.WithRegion("")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), region_config)
	if err != nil {
		log.Fatal(err)
	}
	iam_svc := iam.NewFromConfig(cfg)

	var role_name = get_role_name(cfg)

	tags_resp, err := iam_svc.ListRoleTags(context.TODO(), &iam.ListRoleTagsInput{
		RoleName: &role_name,
	})

	if err != nil {
		log.Fatal(err)
	}

	var newEnv []string = os.Environ()

	for _, v := range tags_resp.Tags {
		if strings.HasPrefix(*v.Key, "secrets_") {
			log.Println("Loading secrets for", *v.Value)
			var fetched_secrets interface{} = fetch_secrets_from_path(*v.Value, cfg)
			newEnv = append(newEnv, flatten_json(fetched_secrets)...)
		}
	}

	log.Printf("Executing %v", os.Args)
	path, err := exec.LookPath(os.Args[1])

	if err != nil {
		log.Fatal(err)
	}

	if err := syscall.Exec(path, os.Args[1:], newEnv); err != nil {
		log.Fatal(err)
	}
}
