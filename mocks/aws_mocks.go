// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -source=main.go -destination=mocks/aws_mocks.go -package aws_mocks
//
// Package aws_mocks is a generated GoMock package.
package aws_mocks

import (
	context "context"
	reflect "reflect"

	iam "github.com/aws/aws-sdk-go-v2/service/iam"
	secretsmanager "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sts "github.com/aws/aws-sdk-go-v2/service/sts"
	gomock "go.uber.org/mock/gomock"
)

// MocksecretsClient is a mock of secretsClient interface.
type MocksecretsClient struct {
	ctrl     *gomock.Controller
	recorder *MocksecretsClientMockRecorder
}

// MocksecretsClientMockRecorder is the mock recorder for MocksecretsClient.
type MocksecretsClientMockRecorder struct {
	mock *MocksecretsClient
}

// NewMocksecretsClient creates a new mock instance.
func NewMocksecretsClient(ctrl *gomock.Controller) *MocksecretsClient {
	mock := &MocksecretsClient{ctrl: ctrl}
	mock.recorder = &MocksecretsClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MocksecretsClient) EXPECT() *MocksecretsClientMockRecorder {
	return m.recorder
}

// GetSecretValue mocks base method.
func (m *MocksecretsClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetSecretValue", varargs...)
	ret0, _ := ret[0].(*secretsmanager.GetSecretValueOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecretValue indicates an expected call of GetSecretValue.
func (mr *MocksecretsClientMockRecorder) GetSecretValue(ctx, params any, optFns ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecretValue", reflect.TypeOf((*MocksecretsClient)(nil).GetSecretValue), varargs...)
}

// MockstsClient is a mock of stsClient interface.
type MockstsClient struct {
	ctrl     *gomock.Controller
	recorder *MockstsClientMockRecorder
}

// MockstsClientMockRecorder is the mock recorder for MockstsClient.
type MockstsClientMockRecorder struct {
	mock *MockstsClient
}

// NewMockstsClient creates a new mock instance.
func NewMockstsClient(ctrl *gomock.Controller) *MockstsClient {
	mock := &MockstsClient{ctrl: ctrl}
	mock.recorder = &MockstsClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockstsClient) EXPECT() *MockstsClientMockRecorder {
	return m.recorder
}

// GetCallerIdentity mocks base method.
func (m *MockstsClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetCallerIdentity", varargs...)
	ret0, _ := ret[0].(*sts.GetCallerIdentityOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCallerIdentity indicates an expected call of GetCallerIdentity.
func (mr *MockstsClientMockRecorder) GetCallerIdentity(ctx, params any, optFns ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCallerIdentity", reflect.TypeOf((*MockstsClient)(nil).GetCallerIdentity), varargs...)
}

// MockiamClient is a mock of iamClient interface.
type MockiamClient struct {
	ctrl     *gomock.Controller
	recorder *MockiamClientMockRecorder
}

// MockiamClientMockRecorder is the mock recorder for MockiamClient.
type MockiamClientMockRecorder struct {
	mock *MockiamClient
}

// NewMockiamClient creates a new mock instance.
func NewMockiamClient(ctrl *gomock.Controller) *MockiamClient {
	mock := &MockiamClient{ctrl: ctrl}
	mock.recorder = &MockiamClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockiamClient) EXPECT() *MockiamClientMockRecorder {
	return m.recorder
}

// ListRoleTags mocks base method.
func (m *MockiamClient) ListRoleTags(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListRoleTags", varargs...)
	ret0, _ := ret[0].(*iam.ListRoleTagsOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoleTags indicates an expected call of ListRoleTags.
func (mr *MockiamClientMockRecorder) ListRoleTags(ctx, params any, optFns ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoleTags", reflect.TypeOf((*MockiamClient)(nil).ListRoleTags), varargs...)
}
