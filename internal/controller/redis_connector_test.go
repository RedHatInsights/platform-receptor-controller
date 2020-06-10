package controller

import (
	"reflect"
	"testing"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/utils"
	"github.com/go-playground/assert/v2"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
)

var testHost string = utils.GetHostname()

func newTestRedisClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})
}

func TestRegisterWithRedis(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	tests := []struct {
		account  string
		nodeID   string
		hostname string
		err      error
	}{
		{account: "01", nodeID: "node-a", hostname: testHost, err: nil},
		{account: "01", nodeID: "node-b", hostname: testHost, err: nil},
		{account: "01", nodeID: "node-b", hostname: testHost, err: DuplicateConnectionError{}},
	}

	for _, tc := range tests {
		got := RegisterWithRedis(c, tc.account, tc.nodeID)
		if got != tc.err {
			t.Fatalf("expected: %v, got: %v", tc.err, got)
		}
		s.CheckGet(t, tc.account+":"+tc.nodeID, tc.hostname)
	}
}

func TestUnregisterWithRedis(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	c.Set("01:node-a", testHost, 0)
	c.Set("01:node-b", testHost, 0)
	c.SAdd("connections", "01:node-a:"+testHost, "01:node-b:"+testHost)

	tests := []struct {
		account  string
		nodeID   string
		expected []string
	}{
		{account: "01", nodeID: "node-a", expected: []string{"01:node-b:" + testHost}},
		{account: "01", nodeID: "node-not-found", expected: []string{"01:node-b:" + testHost}},
		{account: "01", nodeID: "node-b", expected: []string{}},
	}

	for _, tc := range tests {
		UnregisterWithRedis(c, tc.account, tc.nodeID)
		if got := c.SMembers("connections"); !reflect.DeepEqual(got.Val(), tc.expected) {
			t.Fatalf("expected: %v, got: %v", tc.expected, got)
		}
	}
}

func TestUpdatedIndexesOnRegister(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	tests := []struct {
		account   string
		nodeID    string
		hostname  string
		accIndex  []string
		connIndex []string
		podIndex  []string
		err       error
	}{
		{
			account:   "01",
			nodeID:    "node-a",
			hostname:  testHost,
			accIndex:  []string{"node-a:" + testHost},
			connIndex: []string{"01:node-a:" + testHost},
			podIndex:  []string{"01:node-a"},
			err:       nil,
		},
		{
			account:   "01",
			nodeID:    "node-b",
			hostname:  testHost,
			accIndex:  []string{"node-a:" + testHost, "node-b:" + testHost},
			connIndex: []string{"01:node-a:" + testHost, "01:node-b:" + testHost},
			podIndex:  []string{"01:node-a", "01:node-b"},
			err:       nil,
		},
		{
			account:   "02",
			nodeID:    "node-a",
			hostname:  testHost,
			accIndex:  []string{"node-a:" + testHost},
			connIndex: []string{"01:node-a:" + testHost, "01:node-b:" + testHost, "02:node-a:" + testHost},
			podIndex:  []string{"01:node-a", "01:node-b", "02:node-a"},
			err:       nil,
		},
	}

	for _, tc := range tests {
		got := RegisterWithRedis(c, tc.account, tc.nodeID)
		if got != tc.err {
			t.Fatalf("expected: %v, got: %v", tc.err, got)
		}
		s.CheckSet(t, tc.account, tc.accIndex...)
		s.CheckSet(t, "connections", tc.connIndex...)
		s.CheckSet(t, testHost, tc.podIndex...)
	}
}

func TestUpdatedIndexesOnUnregister(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	c.SAdd("01", "node-a:"+testHost, "node-b:"+testHost)
	c.SAdd("connections", "01:node-a:"+testHost, "01:node-b:"+testHost)
	c.SAdd(testHost, "01:node-a", "01:node-b")

	tests := []struct {
		account string
		nodeID  string
	}{
		{account: "01", nodeID: "node-a"},
		{account: "01", nodeID: "node-b"},
	}

	for _, tc := range tests {
		UnregisterWithRedis(c, tc.account, tc.nodeID)
	}

	assert.Equal(t, c.SMembers("01").Val(), []string{})
	assert.Equal(t, c.SMembers("connections").Val(), []string{})
	assert.Equal(t, c.SMembers(testHost).Val(), []string{})
}
