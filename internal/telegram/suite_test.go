package telegram

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTelegram(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Telegram Suite")
}
