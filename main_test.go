package main

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	aws_mocks "slicelife/fetch-secrets/mocks"
	"testing"
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

	assert.Equal(s.T(), "", result)
	assert.Error(s.T(), err)
}

func (s *FetchSecretsTestSuite) TestGetRole_InvalidARN() {

	badARNVal := "not-today"
	badARN := &sts.GetCallerIdentityOutput{
		Arn: &badARNVal,
	}

	s.stsMock.EXPECT().GetCallerIdentity(s.ctx, &sts.GetCallerIdentityInput{}).Return(badARN, nil)
	result, err := getRole(s.ctx, s.stsMock)

	assert.Equal(s.T(), "", result)
	assert.Error(s.T(), err)
}

func (s *FetchSecretsTestSuite) TestGetRole_EmptyARN() {

	emptyARNVal := "one/"
	emptyARN := &sts.GetCallerIdentityOutput{
		Arn: &emptyARNVal,
	}

	s.stsMock.EXPECT().GetCallerIdentity(s.ctx, &sts.GetCallerIdentityInput{}).Return(emptyARN, nil)
	result, err := getRole(s.ctx, s.stsMock)

	assert.Equal(s.T(), "", result)
	assert.Error(s.T(), err)
}

func (s *FetchSecretsTestSuite) TestGetRole_WildCard() {

	wildcardARNVal := "one/*"
	wildcardARN := &sts.GetCallerIdentityOutput{
		Arn: &wildcardARNVal,
	}

	s.stsMock.EXPECT().GetCallerIdentity(s.ctx, &sts.GetCallerIdentityInput{}).Return(wildcardARN, nil)
	result, err := getRole(s.ctx, s.stsMock)

	assert.Equal(s.T(), "", result)
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

func TestFetchSecrets(t *testing.T) {
	suite.Run(t, new(FetchSecretsTestSuite))
}
