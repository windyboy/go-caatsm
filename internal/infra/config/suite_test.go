package config

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInfraConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Infra Config Suite")
}
