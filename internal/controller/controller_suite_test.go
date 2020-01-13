package controller_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = AfterSuite(func() {
	Eventually(leaks).ShouldNot(HaveOccurred())
})

func leaks() error {
	return goleak.Find(goleak.IgnoreTopFunction("github.com/onsi/ginkgo/internal/specrunner.(*SpecRunner).registerForInterrupts"))
}
