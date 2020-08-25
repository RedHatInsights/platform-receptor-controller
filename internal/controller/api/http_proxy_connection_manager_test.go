package api

import (
	"context"
	"testing"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"

	"github.com/alicebob/miniredis"
	"github.com/go-playground/assert/v2"
	"github.com/go-redis/redis"
)

func newTestRedisClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})
}

func TestGetConnection(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	locator := &RedisConnectionLocator{
		Client: c,
		Cfg:    config.GetConfig(),
	}

	_ = controller.RegisterWithRedis(locator.Client, "01", "node-a", "localhost")

	tests := []struct {
		account      string
		nodeID       string
		expectedConn controller.Receptor
	}{
		{
			account: "01",
			nodeID:  "node-a",
			expectedConn: &ReceptorHttpProxy{
				Hostname:      "localhost",
				AccountNumber: "01",
				NodeID:        "node-a",
				Config:        locator.Cfg,
			},
		},
		{
			account:      "01",
			nodeID:       "not-found",
			expectedConn: nil,
		},
	}

	for _, tc := range tests {
		conn := locator.GetConnection(context.TODO(), tc.account, tc.nodeID)
		assert.Equal(t, conn, tc.expectedConn)
	}
}

func TestGetConnectionsByAccount(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	locator := &RedisConnectionLocator{
		Client: c,
		Cfg:    config.GetConfig(),
	}

	_ = controller.RegisterWithRedis(c, "01", "node-a", "localhost")
	_ = controller.RegisterWithRedis(c, "01", "node-b", "localhost")
	_ = controller.RegisterWithRedis(c, "02", "node-c", "localhost")

	tests := []struct {
		account       string
		expectedConns map[string]controller.Receptor
	}{
		{
			account: "01",
			expectedConns: map[string]controller.Receptor{
				"node-a": &ReceptorHttpProxy{
					Hostname:      "localhost",
					AccountNumber: "01",
					NodeID:        "node-a",
					Config:        locator.Cfg,
				},
				"node-b": &ReceptorHttpProxy{
					Hostname:      "localhost",
					AccountNumber: "01",
					NodeID:        "node-b",
					Config:        locator.Cfg,
				},
			},
		},
		{
			account: "02",
			expectedConns: map[string]controller.Receptor{
				"node-c": &ReceptorHttpProxy{
					Hostname:      "localhost",
					AccountNumber: "02",
					NodeID:        "node-c",
					Config:        locator.Cfg,
				},
			},
		},
		{
			account:       "not-found",
			expectedConns: map[string]controller.Receptor{},
		},
	}

	for _, tc := range tests {
		res := locator.GetConnectionsByAccount(context.TODO(), tc.account)
		assert.Equal(t, res, tc.expectedConns)
	}
}

func TestGetAllConnections(t *testing.T) {
	s, _ := miniredis.Run()
	defer s.Close()

	c := newTestRedisClient(s.Addr())

	locator := &RedisConnectionLocator{
		Client: c,
		Cfg:    config.GetConfig(),
	}

	_ = controller.RegisterWithRedis(c, "01", "node-a", "localhost")
	_ = controller.RegisterWithRedis(c, "01", "node-b", "localhost")
	_ = controller.RegisterWithRedis(c, "02", "node-c", "localhost")

	res := locator.GetAllConnections(context.TODO())

	assert.Equal(t, map[string]map[string]controller.Receptor{
		"01": {
			"node-a": &ReceptorHttpProxy{
				Hostname:      "localhost",
				AccountNumber: "01",
				NodeID:        "node-a",
				Config:        locator.Cfg,
			},
			"node-b": &ReceptorHttpProxy{
				Hostname:      "localhost",
				AccountNumber: "01",
				NodeID:        "node-b",
				Config:        locator.Cfg,
			},
		},
		"02": {
			"node-c": &ReceptorHttpProxy{
				Hostname:      "localhost",
				AccountNumber: "02",
				NodeID:        "node-c",
				Config:        locator.Cfg,
			},
		},
	}, res)
}
