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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	timeout         = 60 * time.Second
	secretTagPrefix = "secret_"
)

type awsClients struct {
	secretsClient secretsClient
	stsClient     stsClient
	iamClient     iamClient
}

var clients *awsClients

//go:generate mockgen -source=main.go -destination=mocks/aws_mocks.go -package aws_mocks

type secretsClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type stsClient interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

type iamClient interface {
	ListRoleTags(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error)
}

func fetchSecretsByID(ctx context.Context, secretsClient secretsClient, secretID string) ([]string, error) {
	result, err := secretsClient.GetSecretValue(ctx,
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretID),
		})
	if err != nil {
		return nil, fmt.Errorf("unable to get secret %q from SecretsManager: %w", secretID, err)
	}

	var secretsJSON map[string]interface{}
	err = json.Unmarshal([]byte(*result.SecretString), &secretsJSON)
	if err != nil {
		return nil, fmt.Errorf("unable unmarshal %q JSON from SecretManager: %w", secretID, err)
	}

	secrets := make([]string, len(secretsJSON))
	for k, v := range secretsJSON {
		fmt.Println("finding ")
		secrets = append(secrets, fmt.Sprintf("%s=%s", k, v))
	}

	return secrets, nil
}

func getRole(ctx context.Context, stsClient stsClient) (string, error) {
	callerID, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("unable to get STS caller ID: %w", err)
	}

	roleParts := strings.Split(*callerID.Arn, "/")
	if len(roleParts) < 2 || roleParts[1] == "" || roleParts[1] == "*" {
		return "", fmt.Errorf("unable to determine role name from arn: %s", *callerID.Arn)
	}

	return roleParts[1], nil
}

func getTags(ctx context.Context, iamClient iamClient, role string) ([]string, error) {
	resp, err := iamClient.ListRoleTags(ctx,
		&iam.ListRoleTagsInput{
			RoleName: &role,
		})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get tags for role: %s", role)
	}

	tags := make([]string, len(resp.Tags))
	for _, t := range resp.Tags {
		tag := *t.Key
		if strings.HasPrefix(tag, secretTagPrefix) {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

func main() {

	path, err := exec.LookPath(os.Args[1])
	if err != nil {
		log.Fatal(fmt.Errorf("executable not found: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(os.Getenv("FS_REGION")))
	if err != nil {
		log.Fatal(fmt.Errorf("unable to load AWS config: %w", err))
	}

	clients = &awsClients{
		secretsClient: secretsmanager.NewFromConfig(cfg),
		stsClient:     sts.NewFromConfig(cfg),
		iamClient:     iam.NewFromConfig(cfg),
	}

	secrets, err := getSecrets(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled {
		log.Fatal(fmt.Errorf("exceeded timeout %s", timeout))
	}

	newEnv := append(os.Environ(), secrets...)
	log.Printf("Executing %v", os.Args)
	if err := syscall.Exec(path, os.Args[1:], newEnv); err != nil {
		log.Fatal(err)
	}
}

func getSecrets(ctx context.Context) ([]string, error) {

	var secrets []string
	role, err := getRole(ctx, clients.stsClient)
	if err != nil {
		return secrets, err
	}

	secretTags, err := getTags(ctx, clients.iamClient, role)
	if err != nil {
		return secrets, err
	}

	for _, tag := range secretTags {
		fetched, err := fetchSecretsByID(ctx, clients.secretsClient, tag)
		if err != nil {
			log.Println(err.Error()) // Non-fatal, allows secret fetching to continue if secret not found or bad JSON
			continue
		}
		secrets = append(secrets, fetched...)
	}

	return secrets, nil
}
