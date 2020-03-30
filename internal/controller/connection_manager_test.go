package controller

import (
	"context"
	"testing"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func init() {
	logger.InitLogger()
}

// FIXME: Move this to a "central" place
type MockReceptor struct {
}

func (mr *MockReceptor) SendMessage(context.Context, string, []string, interface{}, string) (*uuid.UUID, error) {
	return nil, nil
}

func (mr *MockReceptor) Ping(context.Context, string, []string) (interface{}, error) {
	return nil, nil
}

func (mr *MockReceptor) Close() {
}

func (mr *MockReceptor) DisconnectReceptorNetwork() {
}

func (mr *MockReceptor) GetCapabilities() interface{} {
	return nil
}

func TestCheckForConnectionThatDoesNotExist(t *testing.T) {
	cm := NewConnectionManager()
	receptorConnection := cm.GetConnection("not gonna find me", "or me")
	if receptorConnection != nil {
		t.Fatalf("Expected to not find a connection, but a connection was found")
	}
}

func TestCheckForConnectionThatDoesNotExistButAccountExists(t *testing.T) {
	registeredAccount := "123"
	cm := NewConnectionManager()
	cm.Register(registeredAccount, "456", &MockReceptor{})
	receptorConnection := cm.GetConnection(registeredAccount, "not gonna find me")
	if receptorConnection != nil {
		t.Fatalf("Expected to not find a connection, but a connection was found")
	}
}

func TestCheckForConnectionThatDoesExist(t *testing.T) {
	mockReceptor := &MockReceptor{}
	cm := NewConnectionManager()
	cm.Register("123", "456", mockReceptor)
	receptorConnection := cm.GetConnection("123", "456")
	if receptorConnection == nil {
		t.Fatalf("Expected to find a connection, but did not find a connection")
	}

	if mockReceptor != receptorConnection {
		t.Fatalf("Found the wrong connection")
	}
}

func TestRegisterAndUnregisterMultipleConnectionsPerAccount(t *testing.T) {
	accountNumber := "0000001"
	var testReceptors = []struct {
		account  string
		node_id  string
		receptor *MockReceptor
	}{
		{accountNumber, "node-a", &MockReceptor{}},
		{accountNumber, "node-b", &MockReceptor{}},
	}
	cm := NewConnectionManager()
	for _, r := range testReceptors {
		cm.Register(r.account, r.node_id, r.receptor)

		actualReceptor := cm.GetConnection(r.account, r.node_id)
		if actualReceptor == nil {
			t.Fatalf("Expected to find a connection, but did not find a connection")
		}

		if r.receptor != actualReceptor {
			t.Fatalf("Found the wrong connection")
		}
	}

	for _, r := range testReceptors {
		cm.Unregister(r.account, r.node_id)
	}
}

func TestUnregisterConnectionThatDoesNotExist(t *testing.T) {
	cm := NewConnectionManager()
	cm.Unregister("not gonna find me", "or me")
}

func TestGetConnectionsByAccount(t *testing.T) {
	accountNumber := "0000001"
	var testReceptors = []struct {
		account  string
		node_id  string
		receptor *MockReceptor
	}{
		{accountNumber, "node-a", &MockReceptor{}},
		{accountNumber, "node-b", &MockReceptor{}},
	}
	cm := NewConnectionManager()
	for _, r := range testReceptors {
		cm.Register(r.account, r.node_id, r.receptor)
	}

	receptorMap := cm.GetConnectionsByAccount(accountNumber)
	if len(receptorMap) != len(testReceptors) {
		t.Fatalf("Expected to find %d connections, but found %d connections", len(testReceptors), len(receptorMap))
	}
}

func TestGetConnectionsByAccountWithNoRegisteredReceptors(t *testing.T) {
	cm := NewConnectionManager()
	receptorMap := cm.GetConnectionsByAccount("0000001")
	if len(receptorMap) != 0 {
		t.Fatalf("Expected to find 0 connections, but found %d connections", len(receptorMap))
	}
}

func TestGetAllConnections(t *testing.T) {

	var testReceptors = map[string]map[string]Receptor{
		"0000001": {"node-a": &MockReceptor{},
			"node-b": &MockReceptor{}},
		"0000002": {"node-a": &MockReceptor{}},
		"0000003": {"node-a": &MockReceptor{}},
	}
	cm := NewConnectionManager()
	for account, receptorMap := range testReceptors {
		for nodeID, receptor := range receptorMap {
			cm.Register(account, nodeID, receptor)
		}
	}

	receptorMap := cm.GetAllConnections()

	if cmp.Equal(testReceptors, receptorMap) != true {
		t.Fatalf("Excepted receptor map and actual receptor map do not match.  Excpected %+v, Actual %+v",
			testReceptors, receptorMap)
	}
}

func TestGetAllConnectionsWithNoRegisteredReceptors(t *testing.T) {
	cm := NewConnectionManager()
	receptorMap := cm.GetAllConnections()
	if len(receptorMap) != 0 {
		t.Fatalf("Expected to find 0 connections, but found %d connections", len(receptorMap))
	}
}
