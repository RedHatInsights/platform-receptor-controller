package controller

import (
	"context"
	"testing"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/utils"
	"github.com/go-playground/assert/v2"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/alicebob/miniredis"
)

func init() {
	logger.InitLogger()
}

var hostname string = utils.GetHostname()

func TestRegisterWithGatewayConnectionManager(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())
	lcm := NewLocalConnectionManager()

	acrf := NewActiveConnectionRegistrarFactory(c, hostname)
	gcm := NewGatewayConnectionRegistrar(c, lcm, acrf, hostname)

	tests := []struct {
		account string
		nodeID  string
		client  Receptor
		err     error
	}{
		{
			account: "01",
			nodeID:  "node-a",
			client:  &MockReceptor{NodeID: "node-a"},
			err:     nil,
		},
		{
			account: "01",
			nodeID:  "node-b",
			client:  &MockReceptor{NodeID: "node-b"},
			err:     nil,
		},
		{
			account: "02",
			nodeID:  "node-a",
			client:  &MockReceptor{NodeID: "node-a"},
			err:     nil,
		},
	}

	for _, tc := range tests {
		got := gcm.Register(context.TODO(), tc.account, tc.nodeID, tc.client)
		if got != tc.err {
			t.Fatalf("expected: %v, got: %v", tc.err, got)
		}
		assert.Equal(t, c.Get(tc.account+":"+tc.nodeID).Val(), hostname)                     // check redis
		assert.Equal(t, tc.client, lcm.GetConnection(context.TODO(), tc.account, tc.nodeID)) // check local connections
	}
}

func TestRegisterDuplicateWithGatewayConnectionManager(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())
	lcm := NewLocalConnectionManager()

	acrf := NewActiveConnectionRegistrarFactory(c, hostname)
	gcm := NewGatewayConnectionRegistrar(c, lcm, acrf, hostname)

	_ = RegisterWithRedis(c, "01", "node-c", hostname)
	lcm.Register(context.TODO(), "01", "node-d", &MockReceptor{NodeID: "node-d"})

	tests := []struct {
		account        string
		nodeID         string
		client         Receptor
		expectedClient Receptor
		expectedHost   string
		err            error
	}{
		{
			account:        "01",
			nodeID:         "node-a",
			client:         &MockReceptor{NodeID: "node-a"},
			expectedClient: &MockReceptor{NodeID: "node-a"},
			expectedHost:   hostname,
			err:            nil,
		},
		{
			account:        "01",
			nodeID:         "node-b",
			client:         &MockReceptor{NodeID: "node-b"},
			expectedClient: &MockReceptor{NodeID: "node-b"},
			expectedHost:   hostname,
			err:            nil,
		},
		// connection registered in redis but not locally
		{
			account:        "01",
			nodeID:         "node-c",
			client:         &MockReceptor{NodeID: "node-c"},
			expectedClient: nil,
			expectedHost:   hostname,
			err:            DuplicateConnectionError{},
		},
		// connection registered locally but not in redis
		{
			account:        "01",
			nodeID:         "node-d",
			client:         &MockReceptor{NodeID: "node-d"},
			expectedClient: nil,
			expectedHost:   "",
			err:            DuplicateConnectionError{},
		},
	}

	for _, tc := range tests {
		got := gcm.Register(context.TODO(), tc.account, tc.nodeID, tc.client)
		if got != tc.err {
			t.Fatalf("expected: %v, got: %v", tc.err, got)
		}
		assert.Equal(t, c.Get(tc.account+":"+tc.nodeID).Val(), tc.expectedHost)
		assert.Equal(t, lcm.GetConnection(context.TODO(), tc.account, tc.nodeID), tc.expectedClient)
	}
}

func TestUnregisterWithGatewayConnectionManager(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())
	lcm := NewLocalConnectionManager()

	acrf := NewActiveConnectionRegistrarFactory(c, hostname)
	gcm := NewGatewayConnectionRegistrar(c, lcm, acrf, hostname)

	_ = gcm.Register(context.TODO(), "01", "node-a", &MockReceptor{NodeID: "node-a"})
	_ = gcm.Register(context.TODO(), "01", "node-b", &MockReceptor{NodeID: "node-b"})
	_ = gcm.Register(context.TODO(), "01", "node-c", &MockReceptor{NodeID: "node-c"})
	_ = gcm.Register(context.TODO(), "01", "node-d", &MockReceptor{NodeID: "node-d"})

	tests := []struct {
		account string
		nodeID  string
		client  Receptor
		err     error
	}{
		{
			account: "01",
			nodeID:  "node-a",
			client:  &MockReceptor{NodeID: "node-a"},
			err:     nil,
		},
		{
			account: "01",
			nodeID:  "node-b",
			client:  &MockReceptor{NodeID: "node-b"},
			err:     nil,
		},
		{
			account: "01",
			nodeID:  "node-c",
			client:  &MockReceptor{NodeID: "node-c"},
			err:     nil,
		},
	}

	for _, tc := range tests {
		assert.Equal(t, c.Get(tc.account+":"+tc.nodeID).Val(), hostname)                     // check redis
		assert.Equal(t, lcm.GetConnection(context.TODO(), tc.account, tc.nodeID), tc.client) // check local connections

		gcm.Unregister(context.TODO(), tc.account, tc.nodeID)

		assert.Equal(t, c.Get(tc.account+":"+tc.nodeID).Val(), "")
		assert.Equal(t, lcm.GetConnection(context.TODO(), tc.account, tc.nodeID), nil)
	}

	assert.Equal(t, c.Get("01:node-d").Val(), hostname)
	assert.Equal(t, lcm.GetConnection(context.TODO(), "01", "node-d"), &MockReceptor{NodeID: "node-d"})
}
