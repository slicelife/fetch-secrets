package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	timeout         = 60 * time.Second
	secretTagPrefix = "secrets_"
)

var (
	awsSecretsClient *secretsmanager.Client //nolint:gochecknoglobals
	awsSTSClient     *sts.Client            //nolint:gochecknoglobals
	awsIAMClient     *iam.Client            //nolint:gochecknoglobals
	errGetRole       = errors.New("error getting role")
	errGetSecret     = errors.New("error getting secret")
	errGetTags       = errors.New("error getting tags")
)

func wrapErrGetRole(s string) error {
	return fmt.Errorf("%w : %s", errGetRole, s)
}

func wrapErrGetSecret(s string) error {
	return fmt.Errorf("%w : %s", errGetSecret, s)
}

func wrapErrGetTags(s string) error {
	return fmt.Errorf("%w : %s", errGetTags, s)
}

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
		return nil, wrapErrGetSecret(fmt.Sprintf("unable to get secret %q from SecretsManager: %e", secretID, err))
	}

	var secretsJSON map[string]any
	err = json.Unmarshal([]byte(*result.SecretString), &secretsJSON)
	if err != nil {
		return nil, wrapErrGetSecret(fmt.Sprintf("unable unmarshal %q JSON from SecretManager: %e", secretID, err))
	}

	var secrets []string //nolint:prealloc
	for k, v := range secretsJSON {
		slog.Info(">>> finding secret value for:", slog.String("secret name", k))
		secrets = append(secrets, fmt.Sprintf("%s=%s", k, v))
	}

	return secrets, nil
}

func getRole(ctx context.Context, stsClient stsClient) (string, error) {
	callerID, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", wrapErrGetRole(fmt.Sprintf("unable to get STS caller ID: %e", err))
	}

	roleParts := strings.Split(*callerID.Arn, "/")
	if len(roleParts) < 2 || roleParts[1] == "" || roleParts[1] == "*" {
		return "", wrapErrGetRole(fmt.Sprintf("unable to determine role name from arn: %s", *callerID.Arn))
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
		return nil, wrapErrGetTags(fmt.Sprintf("unable to get tags for role: %s", role))
	}

	var tags []string
	for _, t := range resp.Tags {
		tag := *t.Key
		if strings.HasPrefix(tag, secretTagPrefix) {
			tags = append(tags, *t.Value)
		}
	}

	return tags, nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("fetch-secrets starting >>>")

	// path, err := exec.LookPath(os.Args[1])
	// if err != nil {
	// 	slog.Error("executable not found:", slog.Any("error", err))
	// 	syscall.Exit(1)
	// }

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

	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
		slog.Error("fetch-secrets timeout:", slog.Duration("timeout", timeout))
		syscall.Exit(4)
	}
	slog.Info(">>> adding secrets to env", slog.Int("numOfSecrets", len(secrets)))
	err = createOrUpdateK8sSecret(ctx, secrets)
	if err != nil {
		slog.Error("fetch-secrets failure:", slog.Any("error", err))
		syscall.Exit(3)
	}
	// newEnv := append(os.Environ(), secrets...)
	// slog.Info("executing:", slog.Any("cmd", os.Args))
	// if err := syscall.Exec(path, os.Args[1:], newEnv); err != nil {
	// 	slog.Error("fetch-secrets failure executing:", slog.Any("cmd", os.Args))
	// 	syscall.Exit(5)
	// }
}

func createOrUpdateK8sSecret(ctx context.Context, secrets []string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	namespace := "consumer"
	svcName := "starrocks"
	secretName := svcName + "-secret"

	secretData := make(map[string][]byte)
	for _, secretString := range secrets {
		secretParts := strings.Split(secretString, "=")
		if len(secretParts) != 2 {
			panic("secretParts")
		}
		secretData[secretParts[0]] = []byte(secretParts[1])
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	log.Print(err)
	if secret == nil {
		log.Print("New k8s secret")
		newSecret := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: secretData,
			Type: "Opaque",
		}

		updatedSecret, err := clientset.CoreV1().Secrets(namespace).Create(ctx, &newSecret, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		fmt.Print(updatedSecret)
	} else {
		log.Print("Update k8s secret")
		secret.Data = secretData
		updatedSecret, err := clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		fmt.Print(updatedSecret)
	}

	return nil
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
			slog.Warn("no valid secret value(s) found:", slog.String("tag-name", tag), slog.Any("error", err)) // Non-fatal, if secret not found or bad JSON
			continue
		}
		secrets = append(secrets, fetched...)
	}

	return secrets, nil
}
