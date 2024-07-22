package repository

import (
	"caatsm/internal/domain"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Repositories Suite")
}

var _ = Describe("Repositories", func() {
	Context("Hasura Repository", func() {
		// var repository *HasuraRepository
		var uuid = "uuid"
		// BeforeEach(func() {

		// })
		It("should mutate a parsed message", func() {
			repository := NewHasuraRepo("http://localhost:8080/v1/graphql", "aviation-test")
			parseMessage := &domain.ParsedMessage{
				Uuid:               uuid,
				MessageID:          "message_id",
				DateTime:           "date_time",
				PriorityIndicator:  "priority_indicator",
				PrimaryAddress:     "primary_address",
				SecondaryAddresses: []string{"secondary_addresses"},
				Originator:         "originator",
				OriginatorDateTime: "originator_date_time",
				Category:           "category",
				BodyAndFooter:      "body_and_footer",
				BodyData:           nil,
				ReceivedAt:         time.Now(),
			}
			err := repository.InsertParsedMessage(parseMessage)
			Expect(err).NotTo(HaveOccurred())

		})
	})
})
