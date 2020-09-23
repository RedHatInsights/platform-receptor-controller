package controller

import (
	"testing"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
)

func init() {
	logger.InitLogger()
}

func TestGetTickerInterval(t *testing.T) {
	cfg := config.GetConfig()
	max := cfg.GatewayActiveConnectionRegistrarPollMaxDelay * int(time.Millisecond)
	min := cfg.GatewayActiveConnectionRegistrarPollMinDelay * int(time.Millisecond)
	interval := int(getTickerInterval(cfg))
	if interval < min {
		t.Fatalf("Expected interval to be greater than min: %d", interval)
	}
	if interval > max {
		t.Fatalf("Expected interval to be less than max: %d", interval)
	}
}
