// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/kt/exec/kubectl/types.go

// Package kubectl is a generated GoMock package.
package kubectl

import (
	exec "os/exec"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCliInterface is a mock of CliInterface interface.
type MockCliInterface struct {
	ctrl     *gomock.Controller
	recorder *MockCliInterfaceMockRecorder
}

// MockCliInterfaceMockRecorder is the mock recorder for MockCliInterface.
type MockCliInterfaceMockRecorder struct {
	mock *MockCliInterface
}

// NewMockCliInterface creates a new mock instance.
func NewMockCliInterface(ctrl *gomock.Controller) *MockCliInterface {
	mock := &MockCliInterface{ctrl: ctrl}
	mock.recorder = &MockCliInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCliInterface) EXPECT() *MockCliInterfaceMockRecorder {
	return m.recorder
}

// ApplyDashboardToCluster mocks base method.
func (m *MockCliInterface) ApplyDashboardToCluster() *exec.Cmd {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyDashboardToCluster")
	ret0, _ := ret[0].(*exec.Cmd)
	return ret0
}

// ApplyDashboardToCluster indicates an expected call of ApplyDashboardToCluster.
func (mr *MockCliInterfaceMockRecorder) ApplyDashboardToCluster() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyDashboardToCluster", reflect.TypeOf((*MockCliInterface)(nil).ApplyDashboardToCluster))
}

// PortForward mocks base method.
func (m *MockCliInterface) PortForward(namespace, resource string, remotePort, localPort int) *exec.Cmd {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PortForward", namespace, resource, remotePort, localPort)
	ret0, _ := ret[0].(*exec.Cmd)
	return ret0
}

// PortForward indicates an expected call of PortForward.
func (mr *MockCliInterfaceMockRecorder) PortForward(namespace, resource, remotePort, localPort interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PortForward", reflect.TypeOf((*MockCliInterface)(nil).PortForward), namespace, resource, remotePort, localPort)
}

// PortForwardDashboardToLocal mocks base method.
func (m *MockCliInterface) PortForwardDashboardToLocal(port string) *exec.Cmd {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PortForwardDashboardToLocal", port)
	ret0, _ := ret[0].(*exec.Cmd)
	return ret0
}

// PortForwardDashboardToLocal indicates an expected call of PortForwardDashboardToLocal.
func (mr *MockCliInterfaceMockRecorder) PortForwardDashboardToLocal(port interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PortForwardDashboardToLocal", reflect.TypeOf((*MockCliInterface)(nil).PortForwardDashboardToLocal), port)
}