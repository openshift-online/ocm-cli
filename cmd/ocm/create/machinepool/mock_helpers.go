// Code generated by MockGen. DO NOT EDIT.
// Source: helpers.go
//
// Generated by this command:
//
//	mockgen -source=helpers.go -package=machinepool -destination=mock_helpers.go
//

// Package machinepool is a generated GoMock package.
package machinepool

import (
	reflect "reflect"

	arguments "github.com/openshift-online/ocm-cli/pkg/arguments"
	ocm "github.com/openshift-online/ocm-cli/pkg/ocm"
	gomock "go.uber.org/mock/gomock"
)

// MockFlagSet is a mock of FlagSet interface.
type MockFlagSet struct {
	ctrl     *gomock.Controller
	recorder *MockFlagSetMockRecorder
}

// MockFlagSetMockRecorder is the mock recorder for MockFlagSet.
type MockFlagSetMockRecorder struct {
	mock *MockFlagSet
}

// NewMockFlagSet creates a new mock instance.
func NewMockFlagSet(ctrl *gomock.Controller) *MockFlagSet {
	mock := &MockFlagSet{ctrl: ctrl}
	mock.recorder = &MockFlagSetMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFlagSet) EXPECT() *MockFlagSetMockRecorder {
	return m.recorder
}

// Changed mocks base method.
func (m *MockFlagSet) Changed(flagId string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Changed", flagId)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Changed indicates an expected call of Changed.
func (mr *MockFlagSetMockRecorder) Changed(flagId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Changed", reflect.TypeOf((*MockFlagSet)(nil).Changed), flagId)
}

// CheckOneOf mocks base method.
func (m *MockFlagSet) CheckOneOf(flagName string, options []arguments.Option) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckOneOf", flagName, options)
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckOneOf indicates an expected call of CheckOneOf.
func (mr *MockFlagSetMockRecorder) CheckOneOf(flagName, options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckOneOf", reflect.TypeOf((*MockFlagSet)(nil).CheckOneOf), flagName, options)
}

// MockMachineTypeListGetter is a mock of MachineTypeListGetter interface.
type MockMachineTypeListGetter struct {
	ctrl     *gomock.Controller
	recorder *MockMachineTypeListGetterMockRecorder
}

// MockMachineTypeListGetterMockRecorder is the mock recorder for MockMachineTypeListGetter.
type MockMachineTypeListGetterMockRecorder struct {
	mock *MockMachineTypeListGetter
}

// NewMockMachineTypeListGetter creates a new mock instance.
func NewMockMachineTypeListGetter(ctrl *gomock.Controller) *MockMachineTypeListGetter {
	mock := &MockMachineTypeListGetter{ctrl: ctrl}
	mock.recorder = &MockMachineTypeListGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMachineTypeListGetter) EXPECT() *MockMachineTypeListGetterMockRecorder {
	return m.recorder
}

// GetMachineTypeOptions mocks base method.
func (m *MockMachineTypeListGetter) GetMachineTypeOptions(arg0 ocm.Cluster) ([]arguments.Option, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMachineTypeOptions", arg0)
	ret0, _ := ret[0].([]arguments.Option)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMachineTypeOptions indicates an expected call of GetMachineTypeOptions.
func (mr *MockMachineTypeListGetterMockRecorder) GetMachineTypeOptions(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMachineTypeOptions", reflect.TypeOf((*MockMachineTypeListGetter)(nil).GetMachineTypeOptions), arg0)
}
