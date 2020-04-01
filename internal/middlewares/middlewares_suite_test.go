package middlewares_test

import (
	"testing"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMiddlewares(t *testing.T) {
	RegisterFailHandler(Fail)
	logger.InitLogger()
	RunSpecs(t, "Middlewares Suite")
}
