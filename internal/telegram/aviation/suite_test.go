package aviation

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAviation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aviation Parser Suite")
}
