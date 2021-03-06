// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/atuleu/golang-socketcan (interfaces: RawInterface)

// Package mock_golang_socketcan is a generated GoMock package.
package main

import (
	reflect "reflect"

	golang_socketcan "github.com/atuleu/golang-socketcan"
	gomock "github.com/golang/mock/gomock"
)

// MockRawInterface is a mock of RawInterface interface
type MockRawInterface struct {
	ctrl     *gomock.Controller
	recorder *MockRawInterfaceMockRecorder
}

// MockRawInterfaceMockRecorder is the mock recorder for MockRawInterface
type MockRawInterfaceMockRecorder struct {
	mock *MockRawInterface
}

// NewMockRawInterface creates a new mock instance
func NewMockRawInterface(ctrl *gomock.Controller) *MockRawInterface {
	mock := &MockRawInterface{ctrl: ctrl}
	mock.recorder = &MockRawInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRawInterface) EXPECT() *MockRawInterfaceMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockRawInterface) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockRawInterfaceMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockRawInterface)(nil).Close))
}

// Receive mocks base method
func (m *MockRawInterface) Receive() (golang_socketcan.CanFrame, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Receive")
	ret0, _ := ret[0].(golang_socketcan.CanFrame)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Receive indicates an expected call of Receive
func (mr *MockRawInterfaceMockRecorder) Receive() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Receive", reflect.TypeOf((*MockRawInterface)(nil).Receive))
}

// Send mocks base method
func (m *MockRawInterface) Send(arg0 golang_socketcan.CanFrame) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send
func (mr *MockRawInterfaceMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockRawInterface)(nil).Send), arg0)
}
