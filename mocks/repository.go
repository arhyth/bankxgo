// Code generated by MockGen. DO NOT EDIT.
// Source: repository.go
//
// Generated by this command:
//
//	mockgen -source=repository.go -destination=mocks/repository.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	bankxgo "github.com/arhyth/bankxgo"
	snowflake "github.com/bwmarrin/snowflake"
	decimal "github.com/shopspring/decimal"
	gomock "go.uber.org/mock/gomock"
)

// MockRepository is a mock of Repository interface.
type MockRepository struct {
	ctrl     *gomock.Controller
	recorder *MockRepositoryMockRecorder
}

// MockRepositoryMockRecorder is the mock recorder for MockRepository.
type MockRepositoryMockRecorder struct {
	mock *MockRepository
}

// NewMockRepository creates a new mock instance.
func NewMockRepository(ctrl *gomock.Controller) *MockRepository {
	mock := &MockRepository{ctrl: ctrl}
	mock.recorder = &MockRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRepository) EXPECT() *MockRepositoryMockRecorder {
	return m.recorder
}

// CreateAccount mocks base method.
func (m *MockRepository) CreateAccount(req bankxgo.CreateAccountReq) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAccount", req)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateAccount indicates an expected call of CreateAccount.
func (mr *MockRepositoryMockRecorder) CreateAccount(req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAccount", reflect.TypeOf((*MockRepository)(nil).CreateAccount), req)
}

// CreditUser mocks base method.
func (m *MockRepository) CreditUser(amount decimal.Decimal, userAcct, systemAcct snowflake.ID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreditUser", amount, userAcct, systemAcct)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreditUser indicates an expected call of CreditUser.
func (mr *MockRepositoryMockRecorder) CreditUser(amount, userAcct, systemAcct any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreditUser", reflect.TypeOf((*MockRepository)(nil).CreditUser), amount, userAcct, systemAcct)
}

// GetAcct mocks base method.
func (m *MockRepository) GetAcct(id snowflake.ID) (*bankxgo.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAcct", id)
	ret0, _ := ret[0].(*bankxgo.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAcct indicates an expected call of GetAcct.
func (mr *MockRepositoryMockRecorder) GetAcct(id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAcct", reflect.TypeOf((*MockRepository)(nil).GetAcct), id)
}
