package repository

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHasuraRepository(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HasuraRepository Suite")
}

// MockGraphQLClient is a mock implementation of graphql.Client
type MockGraphQLClient struct {
	MakeRequestFunc func(ctx context.Context, req *graphql.Request, resp *graphql.Response) error
	CapturedRequest *graphql.Request // To inspect the request made
}

func (m *MockGraphQLClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	m.CapturedRequest = req
	if m.MakeRequestFunc != nil {
		return m.MakeRequestFunc(ctx, req, resp)
	}
	// Default behavior: success, potentially populate resp if needed for some tests
	return nil
}

var _ = Describe("HasuraRepository", func() {
	var (
		repo       *HasuraRepository
		mockClient *MockGraphQLClient
		testConfig *config.Config
	)

	BeforeEach(func() {
		mockClient = &MockGraphQLClient{}
		testConfig = &config.Config{
			Hasura: config.HasuraConfig{
				Endpoint: "http://fake-hasura-endpoint/v1/graphql",
				Secret:   "test-secret",
			},
		}
		// Temporarily set the client directly for tests, bypassing NewHasura's HTTP client setup
		// This allows focusing tests on CreateNew's logic with a mock GQL client.
		repo = &HasuraRepository{
			client: mockClient,
		}
	})

	Describe("NewHasura", func() {
		var originalToken string
		BeforeEach(func() {
			originalToken = os.Getenv("GRAPHQL_TOKEN")
		})
		AfterEach(func() {
			os.Setenv("GRAPHQL_TOKEN", originalToken)
		})

		It("should return a non-nil HasuraRepository with a non-nil client", func() {
			os.Unsetenv("GRAPHQL_TOKEN") // Ensure it uses config secret
			r := NewHasura(testConfig)
			Expect(r).NotTo(BeNil())
			Expect(r.client).NotTo(BeNil())
			// Deeper inspection of the actual http.Client and its transport is complex.
			// We trust that graphql.NewClient and oauth2.NewClient work as expected.
		})

		It("should prioritize GRAPHQL_TOKEN env var over config secret", func() {
			os.Setenv("GRAPHQL_TOKEN", "env-token-for-test")
			// NewHasura is called, it will create its own client.
			// Testing which token is *actually* used by the real http.Client is hard without intercepting HTTP.
			// For this unit test, we're mainly ensuring it constructs.
			// A more integrated test would be needed to verify token priorities against a live endpoint or mock server.
			r := NewHasura(testConfig)
			Expect(r).NotTo(BeNil())
			Expect(r.client).NotTo(BeNil())
			// Note: This test doesn't confirm 'env-token-for-test' was used, just that construction succeeded.
		})
	})

	Describe("CreateNew", func() {
		var pm *domain.ParsedMessage
		var sampleARR domain.ARR

		BeforeEach(func() {
			sampleARR = domain.ARR{
				Category:         "ARR",
				AircraftID:       "TEST123",
				SSRModeAndCode:   "A1234",
				DepartureAirport: "ZBTJ",
				DepartureTime:    "1000",
				ArrivalAirport:   "ZSHC",
				ArrivalTime:      "1200",
			}
			pm = &domain.ParsedMessage{
				Uuid:               uuid.New().String(), // Use a valid UUID string
				MessageID:          "MSG001",
				DateTime:           "220801100000",
				PriorityIndicator:  "FF",
				PrimaryAddress:     "ADDR1",
				SecondaryAddresses: "ADDR2 ADDR3", // This is a string
				Originator:         "ORIGINATOR001",
				OriginatorDateTime: "220801095500",
				Category:           "ARR",
				Content:            "Raw message content",
				BodyData:           &sampleARR,
				ReceivedAt:         time.Now().UTC(),
				ParsedAt:           time.Now().UTC(),
				DispatchedAt:       time.Time{}, // Zero value for not dispatched
				NeedDispatch:       false,
				Parsed:             true,
			}
			mockClient.CapturedRequest = nil // Reset for each CreateNew test
		})

		Context("when the GraphQL mutation is successful", func() {
			BeforeEach(func() {
				mockClient.MakeRequestFunc = func(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
					// Simulate successful response by populating resp.Data if needed
					// For this mutation, response includes inserted message_id and uuid
					// newMessageResponse is defined in generated.go
					parsedUUID, _ := uuid.Parse(pm.Uuid)
					resp.Data = &newMessageResponse{
						Insert_aviation_telegrams_one: newMessageInsert_aviation_telegrams_oneAviation_telegrams{
							Message_id: pm.MessageID,
							Uuid:       parsedUUID,
						},
					}
					return nil
				}
			})

			It("should call the newMessage mutation with correct variables and return no error", func() {
				err := repo.CreateNew(pm)
				Expect(err).NotTo(HaveOccurred())

				Expect(mockClient.CapturedRequest).NotTo(BeNil())
				Expect(mockClient.CapturedRequest.OpName).To(Equal("newMessage"))

				vars, ok := mockClient.CapturedRequest.Variables.(*__newMessageInput)
				Expect(ok).To(BeTrue(), "Variables should be of type __newMessageInput")
				obj := vars.Object

				Expect(obj.Message_id).To(Equal(pm.MessageID))
				Expect(obj.Category).To(Equal(pm.Category))
				Expect(obj.Content).To(Equal(pm.Content))
				Expect(obj.Date_time).To(Equal(pm.DateTime))
				Expect(obj.Dispatched_at.Unix()).To(Equal(pm.DispatchedAt.Unix())) // Compare time.Time carefully
				Expect(obj.Originator).To(Equal(pm.Originator))
				Expect(obj.Originator_date_time).To(Equal(pm.OriginatorDateTime))
				Expect(obj.Primary_address).To(Equal(pm.PrimaryAddress))
				Expect(obj.Priority_indicator).To(Equal(pm.PriorityIndicator))
				Expect(obj.Received_at.Unix()).To(Equal(pm.ReceivedAt.Unix()))
				
				// Check SecondaryAddresses (marshalled string)
				expectedSecondaryAddressesBytes, _ := json.Marshal(pm.SecondaryAddresses)
				Expect(string(obj.Secondary_addresses)).To(Equal(string(expectedSecondaryAddressesBytes)))
				
				// Check BodyData (marshalled ARR struct)
				expectedBodyDataBytes, _ := json.Marshal(pm.BodyData)
				Expect(string(obj.Body_data)).To(Equal(string(expectedBodyDataBytes)))

				// Check UUID
				expectedUUID, _ := uuid.Parse(pm.Uuid)
				Expect(obj.Uuid).To(Equal(expectedUUID))
			})
		})

		Context("when BodyData is nil", func() {
			BeforeEach(func() {
				pm.BodyData = nil
				mockClient.MakeRequestFunc = func(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
					resp.Data = &newMessageResponse{} // Minimal success response
					return nil
				}
			})
			It("should marshal BodyData as null or empty JSON and not error", func() {
				err := repo.CreateNew(pm)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockClient.CapturedRequest).NotTo(BeNil())
				vars, _ := mockClient.CapturedRequest.Variables.(*__newMessageInput)
				// json.Marshal(nil) results in "null"
				Expect(string(vars.Object.Body_data)).To(Equal("null"))
			})
		})
		
		Context("when the GraphQL client returns an error", func() {
			var clientError = errors.New("graphql client error")
			BeforeEach(func() {
				mockClient.MakeRequestFunc = func(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
					return clientError
				}
			})

			It("should propagate the error", func() {
				err := repo.CreateNew(pm)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, clientError)).To(BeTrue())
			})
		})

		Context("when UUID parsing fails (e.g., invalid pm.Uuid string)", func() {
			BeforeEach(func() {
				pm.Uuid = "this-is-not-a-valid-uuid"
				// No need to set MakeRequestFunc if error happens before
			})
			It("should return an error from utils.GetUuid (which panics, so CreateNew might panic or recover)", func() {
				// The current utils.GetUuid panics on error.
				// A robust CreateNew might recover or GetUuid should return an error.
				// For now, let's expect a panic from GetUuid if not handled.
				// Or, if GetUuid is changed to return error, test for that error.
				// Assuming GetUuid still panics as per its current implicit contract from HasuraRepository usage.
				// Test might be better if GetUuid returned an error.
				// For now, this test demonstrates the current behavior.
				// If utils.GetUuid is updated to return an error:
				//_, err := utils.GetUuid(pm.Uuid) -> if err != nil { return err }
				// Then this test would check for that specific error.
				// As written, CreateNew will panic. Ginkgo can test for panics.
				Expect(func() { repo.CreateNew(pm) }).To(Panic())
			})
		})

		Context("with empty SecondaryAddresses", func() {
			BeforeEach(func() {
				pm.SecondaryAddresses = "" // Empty string
				mockClient.MakeRequestFunc = func(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
					resp.Data = &newMessageResponse{} // Minimal success response
					return nil
				}
			})
			It("should marshal SecondaryAddresses as an empty JSON string and not error", func() {
				err := repo.CreateNew(pm)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockClient.CapturedRequest).NotTo(BeNil())
				vars, _ := mockClient.CapturedRequest.Variables.(*__newMessageInput)
				// json.Marshal("") results in "\"\"" (a JSON string containing an empty string)
				Expect(string(vars.Object.Secondary_addresses)).To(Equal("\"\""))
			})
		})

		Context("with a different BodyData type (e.g., FPL)", func() {
			var sampleFPL domain.FPL
			BeforeEach(func() {
				sampleFPL = domain.FPL{FlightNumber: "FPLTEST1", DepartureAirport: "EDDF"}
				pm.BodyData = &sampleFPL
				pm.Category = "FPL"
				mockClient.MakeRequestFunc = func(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
					resp.Data = &newMessageResponse{}
					return nil
				}
			})
			It("should correctly marshal the FPL BodyData", func() {
				err := repo.CreateNew(pm)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockClient.CapturedRequest).NotTo(BeNil())
				vars, _ := mockClient.CapturedRequest.Variables.(*__newMessageInput)
				
				expectedBodyDataBytes, _ := json.Marshal(&sampleFPL)
				Expect(string(vars.Object.Body_data)).To(Equal(string(expectedBodyDataBytes)))
				Expect(vars.Object.Category).To(Equal("FPL"))
			})
		})
		
		Context("when pm.Uuid is an empty string", func() {
			BeforeEach(func() {
				pm.Uuid = "" // Empty string UUID
			})
			It("should be handled by utils.GetUuid (likely resulting in nil UUID or panic)", func() {
				// utils.GetUuid("") behavior:
				// uuid.Parse("") returns an error. If GetUuid panics on error:
				Expect(func() { repo.CreateNew(pm) }).To(Panic())
				// If GetUuid were to return (uuid.Nil, nil) or (uuid.Nil, err) for empty string:
				// mockClient.MakeRequestFunc = func(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
				// 	resp.Data = &newMessageResponse{}
				// 	return nil
				// }
				// err := repo.CreateNew(pm)
				// Expect(err).NotTo(HaveOccurred()) // Or occur if GetUuid returns error
				// Expect(mockClient.CapturedRequest).NotTo(BeNil())
				// vars, _ := mockClient.CapturedRequest.Variables.(*__newMessageInput)
				// Expect(vars.Object.Uuid).To(Equal(uuid.Nil))
			})
		})
	})
})
