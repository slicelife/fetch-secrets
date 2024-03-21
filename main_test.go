package main

import (
	"context"
	"errors"
	aws_mocks "slicelife/fetch-secrets/mocks"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type FetchSecretsTestSuite struct {
	suite.Suite
	ctx         context.Context
	ctrl        *gomock.Controller
	stsMock     *aws_mocks.MockstsClient
	iamMock     *aws_mocks.MockiamClient
	secretsMock *aws_mocks.MocksecretsClient
}

func (s *FetchSecretsTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.ctrl = gomock.NewController(s.T())
	s.stsMock = aws_mocks.NewMockstsClient(s.ctrl)
	s.iamMock = aws_mocks.NewMockiamClient(s.ctrl)
	s.secretsMock = aws_mocks.NewMocksecretsClient(s.ctrl)
}

func (s *FetchSecretsTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *FetchSecretsTestSuite) TestGetRole_STSError() {
	s.stsMock.EXPECT().GetCallerIdentity(s.ctx, &sts.GetCallerIdentityInput{}).Return(nil, errors.New("nope"))

	result, err := getRole(s.ctx, s.stsMock)

	assert.Empty(s.T(), result)
	assert.Error(s.T(), err)
}

func (s *FetchSecretsTestSuite) TestGetRole_InvalidARN() {
	badARNVal := "not-today"
	badARN := &sts.GetCallerIdentityOutput{
		Arn: &badARNVal,
	}
	s.stsMock.EXPECT().GetCallerIdentity(s.ctx, &sts.GetCallerIdentityInput{}).Return(badARN, nil)

	result, err := getRole(s.ctx, s.stsMock)

	assert.Empty(s.T(), result)
	assert.Error(s.T(), err)
}

func (s *FetchSecretsTestSuite) TestGetRole_EmptyARN() {
	emptyARNVal := "one/"
	emptyARN := &sts.GetCallerIdentityOutput{
		Arn: &emptyARNVal,
	}
	s.stsMock.EXPECT().GetCallerIdentity(s.ctx, &sts.GetCallerIdentityInput{}).Return(emptyARN, nil)

	result, err := getRole(s.ctx, s.stsMock)

	assert.Empty(s.T(), result)
	assert.Error(s.T(), err)
}

func (s *FetchSecretsTestSuite) TestGetRole_WildCard() {
	wildcardARNVal := "one/*"
	wildcardARN := &sts.GetCallerIdentityOutput{
		Arn: &wildcardARNVal,
	}
	s.stsMock.EXPECT().GetCallerIdentity(s.ctx, &sts.GetCallerIdentityInput{}).Return(wildcardARN, nil)

	result, err := getRole(s.ctx, s.stsMock)

	assert.Empty(s.T(), result)
	assert.Error(s.T(), err)
}

func (s *FetchSecretsTestSuite) TestGetRole_Success() {

	validARNVal := "marco/polo"
	validARN := &sts.GetCallerIdentityOutput{
		Arn: &validARNVal,
	}
	s.stsMock.EXPECT().GetCallerIdentity(s.ctx, &sts.GetCallerIdentityInput{}).Return(validARN, nil)

	result, err := getRole(s.ctx, s.stsMock)

	assert.Equal(s.T(), "polo", result)
	assert.NoError(s.T(), err)
}

func (s *FetchSecretsTestSuite) TestGetTags_IAMError() {
	s.iamMock.EXPECT().ListRoleTags(s.ctx, &iam.ListRoleTagsInput{
		RoleName: stringPtr("validRole"),
	}).Return(&iam.ListRoleTagsOutput{}, errors.New("nope"))

	result, err := getTags(s.ctx, s.iamMock, "validRole")

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "tags")
	assert.Empty(s.T(), result)
}

func (s *FetchSecretsTestSuite) TestGetTags_EmptyTagSetNoErr() {
	s.iamMock.EXPECT().ListRoleTags(s.ctx, &iam.ListRoleTagsInput{
		RoleName: stringPtr("validRole"),
	}).Return(&iam.ListRoleTagsOutput{Tags: []types.Tag{}}, nil)

	result, err := getTags(s.ctx, s.iamMock, "validRole")

	assert.NoError(s.T(), err)
	assert.Empty(s.T(), result)
}

func (s *FetchSecretsTestSuite) TestGetTags_NoSecretTagsNoErr() {
	tags := []types.Tag{
		{
			Key:   stringPtr("tag-key"),
			Value: stringPtr("tag-value"),
		},
	}

	s.iamMock.EXPECT().ListRoleTags(s.ctx, &iam.ListRoleTagsInput{
		RoleName: stringPtr("validRole"),
	}).Return(&iam.ListRoleTagsOutput{Tags: tags}, nil)

	result, err := getTags(s.ctx, s.iamMock, "validRole")

	assert.NoError(s.T(), err)
	assert.Empty(s.T(), result)
}

func (s *FetchSecretsTestSuite) TestGetTags_Success() {
	tags := []types.Tag{
		{
			Key:   stringPtr("tag-key"),
			Value: stringPtr("tag-value"),
		},
		{
			Key:   stringPtr("secrets_tag-key"),
			Value: stringPtr("secrets-tag-value"),
		},
	}

	s.iamMock.EXPECT().ListRoleTags(s.ctx, &iam.ListRoleTagsInput{
		RoleName: stringPtr("validRole"),
	}).Return(&iam.ListRoleTagsOutput{Tags: tags}, nil)

	result, err := getTags(s.ctx, s.iamMock, "validRole")

	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), result)
	assert.Equal(s.T(), len(result), 1)
	assert.Contains(s.T(), result, "secrets-tag-value")
}

func (s *FetchSecretsTestSuite) TestFetchSecretsByID_SecretsError() {
	s.secretsMock.EXPECT().GetSecretValue(s.ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String("super-secret-ID"),
	}).Return(nil, errors.New("nope"))

	result, err := getSecretsByID(s.ctx, s.secretsMock, "super-secret-ID")

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "unable")
	assert.Empty(s.T(), result)
}

func (s *FetchSecretsTestSuite) TestFetchSecretsByID_BadJSON() {
	s.secretsMock.EXPECT().GetSecretValue(s.ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String("super-secret-ID"),
	}).Return(&secretsmanager.GetSecretValueOutput{
		SecretString: stringPtr("bad-json"),
	}, nil)

	result, err := getSecretsByID(s.ctx, s.secretsMock, "super-secret-ID")

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "unmarshal")
	assert.Empty(s.T(), result)
}

func (s *FetchSecretsTestSuite) TestFetchSecretsByID_Success() {
	json := `{
  		"SECRET-1": "secret1-value",
  		"SECRET-2": "secret2-value"
	}`

	s.secretsMock.EXPECT().GetSecretValue(s.ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String("super-secret-ID"),
	}).Return(&secretsmanager.GetSecretValueOutput{
		SecretString: stringPtr(json),
	}, nil)

	result, err := getSecretsByID(s.ctx, s.secretsMock, "super-secret-ID")

	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), result)
	assert.Contains(s.T(), result, "SECRET-1=secret1-value")
	assert.Contains(s.T(), result, "SECRET-2=secret2-value")
}

func TestFetchSecrets(t *testing.T) {
	suite.Run(t, new(FetchSecretsTestSuite))
}

func stringPtr(s string) *string {
	return &s
}
