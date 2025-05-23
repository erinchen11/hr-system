// Code generated by MockGen. DO NOT EDIT.
// Source: internal/interfaces/account_service.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	models "github.com/erinchen11/hr-system/internal/models"
	gomock "github.com/golang/mock/gomock"
	uuid "github.com/google/uuid"
)

// MockAccountService is a mock of AccountService interface.
type MockAccountService struct {
	ctrl     *gomock.Controller
	recorder *MockAccountServiceMockRecorder
}

// MockAccountServiceMockRecorder is the mock recorder for MockAccountService.
type MockAccountServiceMockRecorder struct {
	mock *MockAccountService
}

// NewMockAccountService creates a new mock instance.
func NewMockAccountService(ctrl *gomock.Controller) *MockAccountService {
	mock := &MockAccountService{ctrl: ctrl}
	mock.recorder = &MockAccountServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAccountService) EXPECT() *MockAccountServiceMockRecorder {
	return m.recorder
}

// Authenticate mocks base method.
func (m *MockAccountService) Authenticate(ctx context.Context, email, password string) (*models.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Authenticate", ctx, email, password)
	ret0, _ := ret[0].(*models.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Authenticate indicates an expected call of Authenticate.
func (mr *MockAccountServiceMockRecorder) Authenticate(ctx, email, password interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Authenticate", reflect.TypeOf((*MockAccountService)(nil).Authenticate), ctx, email, password)
}

// ChangePassword mocks base method.
func (m *MockAccountService) ChangePassword(ctx context.Context, accountID uuid.UUID, oldPassword, newPassword string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangePassword", ctx, accountID, oldPassword, newPassword)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangePassword indicates an expected call of ChangePassword.
func (mr *MockAccountServiceMockRecorder) ChangePassword(ctx, accountID, oldPassword, newPassword interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangePassword", reflect.TypeOf((*MockAccountService)(nil).ChangePassword), ctx, accountID, oldPassword, newPassword)
}

// CreateAccountWithEmployment mocks base method.
func (m *MockAccountService) CreateAccountWithEmployment(ctx context.Context, acc *models.Account, emp *models.Employment) (*models.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAccountWithEmployment", ctx, acc, emp)
	ret0, _ := ret[0].(*models.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAccountWithEmployment indicates an expected call of CreateAccountWithEmployment.
func (mr *MockAccountServiceMockRecorder) CreateAccountWithEmployment(ctx, acc, emp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAccountWithEmployment", reflect.TypeOf((*MockAccountService)(nil).CreateAccountWithEmployment), ctx, acc, emp)
}

// GetAccount mocks base method.
func (m *MockAccountService) GetAccount(ctx context.Context, accountID uuid.UUID) (*models.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccount", ctx, accountID)
	ret0, _ := ret[0].(*models.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccount indicates an expected call of GetAccount.
func (mr *MockAccountServiceMockRecorder) GetAccount(ctx, accountID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccount", reflect.TypeOf((*MockAccountService)(nil).GetAccount), ctx, accountID)
}

// GetJobGradeByCode mocks base method.
func (m *MockAccountService) GetJobGradeByCode(ctx context.Context, code string) (*models.JobGrade, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetJobGradeByCode", ctx, code)
	ret0, _ := ret[0].(*models.JobGrade)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetJobGradeByCode indicates an expected call of GetJobGradeByCode.
func (mr *MockAccountServiceMockRecorder) GetJobGradeByCode(ctx, code interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetJobGradeByCode", reflect.TypeOf((*MockAccountService)(nil).GetJobGradeByCode), ctx, code)
}
