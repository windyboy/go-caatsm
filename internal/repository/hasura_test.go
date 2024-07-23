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
	// Context("Just a simple test", func() {

	// 	// Define a struct for the mutation input to match the expected GraphQL input
	// 	// Define a struct for the mutation input to match the expected GraphQL input
	// 	type aviation_user_insert_input struct {
	// 		Name     graphql.String `json:"name"`
	// 		Email    graphql.String `json:"email"`
	// 		UpdateAt graphql.String `json:"update_at"` // Use string for timestamp
	// 	}
	// 	// Get the current time in RFC3339 format
	// 	currentTime := time.Now().Format(time.RFC3339)

	// 	input := aviation_user_insert_input{
	// 		Name:     graphql.String("new"),
	// 		Email:    graphql.String("2@2.com"),
	// 		UpdateAt: graphql.String(currentTime), // Use formatted string
	// 	}

	// 	// Define the mutation
	// 	var mutation struct {
	// 		InsertAviationUserOne struct {
	// 			ID   int `json:"id"`
	// 			Name string
	// 		} `graphql:"insert_aviation_user_one(object: $object)"`
	// 	}

	// 	// Define the mutation variables
	// 	// Define the mutation variables
	// 	variables := map[string]interface{}{
	// 		"object": input,
	// 	}
	// 	// Define the mutation

	// 	client := graphql.NewClient("http://localhost:8080/v1/graphql", nil)

	// 	err := client.Mutate(context.Background(), &mutation, variables)
	// 	It("should not error", func() {
	// 		Expect(err).NotTo(HaveOccurred())
	// 	})

	// 	It("name should be new", func() {
	// 		Expect(mutation.InsertAviationUserOne.Name).To(Equal("new"))
	// 	})

	// })

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
				BodyData:           domain.ARR{AircraftID: "aircraft_id", Category: "ARR", DepartureAirport: "departure_airport", DepartureTime: "departure_time", ArrivalAirport: "arrival_airport", ArrivalTime: "arrival_time"},
				ReceivedAt:         time.Now(),
			}
			err := repository.InsertParsedMessage(parseMessage)
			Expect(err).NotTo(HaveOccurred())

		})
	})
})
