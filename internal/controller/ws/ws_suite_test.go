package ws_test

import (
	"testing"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	logger.InitLogger()
	RunSpecs(t, "Controller Suite")
}
