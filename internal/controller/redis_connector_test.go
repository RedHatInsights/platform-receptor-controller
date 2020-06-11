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

func TestGetRedisConnection(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	c.Set("01:node-a", "localhost", 0)

	tests := []struct {
		account string
		nodeID  string
		want    string
		err     error
	}{
		{account: "01", nodeID: "node-a", want: "localhost", err: nil},
		{account: "01", nodeID: "bad-node", want: "", err: redis.Nil},
	}

	for _, tc := range tests {
		conn, err := GetRedisConnection(c, tc.account, tc.nodeID)
		assert.Equal(t, conn, tc.want)
		assert.Equal(t, err, tc.err)
	}
}

func TestGetRedisConnectionsByAccount(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	c.SAdd("01", "node-a:localhost", "node-b:localhost")
	c.SAdd("02", "node-c:localhost")

	tests := []struct {
		account string
		want    map[string]string
		err     error
	}{
		{account: "01", want: map[string]string{"node-a": "localhost", "node-b": "localhost"}, err: nil},
		{account: "02", want: map[string]string{"node-c": "localhost"}, err: nil},
		{account: "not-found", want: map[string]string{}, err: nil},
	}

	for _, tc := range tests {
		res, err := GetRedisConnectionsByAccount(c, tc.account)
		assert.Equal(t, res, tc.want)
		assert.Equal(t, err, tc.err)
	}
}

func TestGetRedisConnectionsByHost(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	c.SAdd("localhost", "01:node-a", "02:node-b", "01:node-b")
	c.SAdd("gateway-pod-9", "03:node-c")

	tests := []struct {
		hostname string
		want     map[string][]string
		err      error
	}{
		{hostname: "localhost", want: map[string][]string{"01": {"node-a", "node-b"}, "02": {"node-b"}}, err: nil},
		{hostname: "gateway-pod-9", want: map[string][]string{"03": {"node-c"}}, err: nil},
		{hostname: "not-found", want: map[string][]string{}, err: nil},
	}

	for _, tc := range tests {
		res, err := GetRedisConnectionsByHost(c, tc.hostname)
		assert.Equal(t, res, tc.want)
		assert.Equal(t, err, tc.err)
	}
}

func TestGetAllRedisConnections(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	c.SAdd("connections", "01:node-a:localhost", "02:node-b:localhost", "01:node-b:localhost")

	res, err := GetAllRedisConnections(c)
	if err != nil {
		t.Fatalf("error getting all connections: %v", err)
	}

	assert.Equal(t, map[string]map[string]string{
		"01": {"node-a": "localhost", "node-b": "localhost"},
		"02": {"node-b": "localhost"},
	}, res)
}
