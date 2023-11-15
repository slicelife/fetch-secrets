package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

var (
	awsSecretsClient *secretsmanager.Client
	awsSTSClient     *sts.Client
	awsIAMClient     *iam.Client
)

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

func getSecretsByID(ctx context.Context, secretsClient secretsClient, secretID string) ([]string, error) {
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

	var secrets []string
	for k, v := range secretsJSON {
		slog.Info(">>> finding secret value for:", slog.String("secret name", k))
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
	slog.Info(">>> getting prefixed tags for role:", slog.String("prefix", secretTagPrefix), slog.String("role", role))
	resp, err := iamClient.ListRoleTags(ctx,
		&iam.ListRoleTagsInput{
			RoleName: &role,
		})
	if err != nil {
		return nil, fmt.Errorf("unable to get tags for role: %s", role)
	}

	var tags []string
	for _, t := range resp.Tags {
		tag := *t.Key
		if strings.HasPrefix(tag, secretTagPrefix) {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

func main() {

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("fetch-secrets starting >>>")

	path, err := exec.LookPath(os.Args[1])
	if err != nil {
		slog.Error("executable not found:", slog.Any("error", err))
		syscall.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(os.Getenv("FS_REGION")))
	if err != nil {
		slog.Error("unable to load AWS config:", slog.Any("error", err))
		syscall.Exit(2)
	}

	awsSecretsClient = secretsmanager.NewFromConfig(cfg)
	awsSTSClient = sts.NewFromConfig(cfg)
	awsIAMClient = iam.NewFromConfig(cfg)

	secrets, err := getSecrets(ctx)
	if err != nil {
		slog.Error("fetch-secrets failure:", slog.Any("error", err))
		syscall.Exit(3)
	}

	if ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled {
		slog.Error("fetch-secrets timeout:", slog.Duration("timeout", timeout))
		syscall.Exit(4)
	}
	slog.Info(">>> adding secrets to env", slog.Int("numOfSecrets", len(secrets)))

	newEnv := append(os.Environ(), secrets...)
	slog.Info("executing:", slog.Any("cmd", os.Args))
	if err := syscall.Exec(path, os.Args[1:], newEnv); err != nil {
		slog.Error("fetch-secrets failure executing:", slog.Any("cmd", os.Args))
		syscall.Exit(5)
	}
}

func getSecrets(ctx context.Context) ([]string, error) {
	var secrets []string
	role, err := getRole(ctx, awsSTSClient)
	if err != nil || role == "" {
		return secrets, err
	}

	secretTags, err := getTags(ctx, awsIAMClient, role)
	if err != nil {
		return secrets, err
	}

	for _, tag := range secretTags {
		fetched, err := getSecretsByID(ctx, awsSecretsClient, tag)
		if err != nil {
			slog.Warn("", slog.Any("fetch-secrets", err)) // Non-fatal, if secret not found or bad JSON
			continue
		}
		secrets = append(secrets, fetched...)
	}

	return secrets, nil
}
