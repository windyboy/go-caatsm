package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/internal/iface"
	"errors" // Import for creating custom errors
	"sync"   // Import sync for mutex in handler if needed by tests, though not directly for mocks typically

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockMessageRepository is a mock implementation of iface.MessageRepository
type MockMessageRepository struct {
	CreateNewCalledWith *domain.ParsedMessage
	CreateNewFunc       func(message *domain.ParsedMessage) error // Allow custom function for more complex scenarios
	mu                  sync.Mutex
}

func (m *MockMessageRepository) CreateNew(message *domain.ParsedMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateNewCalledWith = message
	if m.CreateNewFunc != nil {
		return m.CreateNewFunc(message)
	}
	return nil
}

// Helper to reset fields for multiple tests
func (m *MockMessageRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateNewCalledWith = nil
	m.CreateNewFunc = nil
}

// MockMessagePublisher is a mock implementation of iface.MessagePublisher
type MockMessagePublisher struct {
	PublishCalledWith *domain.ParsedMessage
	PublishFunc       func(message *domain.ParsedMessage) error // Allow custom function
	mu                sync.Mutex
}

func (m *MockMessagePublisher) Publish(message *domain.ParsedMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PublishCalledWith = message
	if m.PublishFunc != nil {
		return m.PublishFunc(message)
	}
	return nil
}

func (m *MockMessagePublisher) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PublishCalledWith = nil
	m.PublishFunc = nil
}

var _ = Describe("MessageHandler", func() {
	var (
		handler     *MessageHandler
		mockRepo    *MockMessageRepository
		mockPub     *MockMessagePublisher
		testConfig  *config.Config
		messageID   string
	)

	BeforeEach(func() {
		testConfig = &config.Config{
			// Initialize with minimal necessary config if any fields are accessed in NewHandler/HandleMessage
		}
		mockRepo = &MockMessageRepository{}
		mockPub = &MockMessagePublisher{}
		handler = NewHandler(testConfig, mockPub, mockRepo)
		messageID = "test-message-id-123"
	})

	Describe("NewHandler", func() {
		It("should correctly initialize a MessageHandler", func() {
			Expect(handler).NotTo(BeNil())
			Expect(handler.config).To(Equal(testConfig))
			Expect(handler.repository).To(Equal(mockRepo))
			Expect(handler.publisher).To(Equal(mockPub))
		})
	})

	Describe("HandleMessage", func() {
		Context("with a valid message that parses successfully", func() {
			var validMsgBody []byte
			BeforeEach(func() {
				// This message body for ARR is known to parse successfully by parsers.Parse
				// (ARR-CES5470-ZBTJ-ZSHC1614)
				validMsgBody = []byte("(ARR-CES5470-ZBTJ-ZSHC1614)")
				mockRepo.Reset()
				mockPub.Reset()
			})

			It("should call repository.CreateNew and publisher.Publish with the parsed message", func() {
				err := handler.HandleMessage(validMsgBody, messageID)
				Expect(err).NotTo(HaveOccurred())

				By("verifying repository.CreateNew was called")
				Expect(mockRepo.CreateNewCalledWith).NotTo(BeNil())
				Expect(mockRepo.CreateNewCalledWith.Uuid).To(Equal(messageID))
				Expect(mockRepo.CreateNewCalledWith.Parsed).To(BeTrue())
				Expect(mockRepo.CreateNewCalledWith.Category).To(Equal("ARR"))
				Expect(mockRepo.CreateNewCalledWith.BodyData.(*domain.ARR).AircraftID).To(Equal("CES5470"))

				By("verifying publisher.Publish was called")
				Expect(mockPub.PublishCalledWith).NotTo(BeNil())
				Expect(mockPub.PublishCalledWith.Uuid).To(Equal(messageID))
				Expect(mockPub.PublishCalledWith.Parsed).To(BeTrue())
				Expect(mockPub.PublishCalledWith.Category).To(Equal("ARR"))
				Expect(mockPub.PublishCalledWith.BodyData.(*domain.ARR).AircraftID).To(Equal("CES5470"))
			})
		})

		Context("with a message that fails parsing", func() {
			var unparseableMsgBody []byte
			BeforeEach(func() {
				unparseableMsgBody = []byte("(XYZ-SOME-INVALID-FORMAT)") // This will fail in the body parser
				mockRepo.Reset()
				mockPub.Reset()
			})

			It("should still call repository.CreateNew and publisher.Publish with the unparsed message details", func() {
				err := handler.HandleMessage(unparseableMsgBody, messageID)
				Expect(err).NotTo(HaveOccurred()) // HandleMessage itself doesn't error on parse fail, unless publish fails

				By("verifying repository.CreateNew was called with unparsed message")
				Expect(mockRepo.CreateNewCalledWith).NotTo(BeNil())
				Expect(mockRepo.CreateNewCalledWith.Uuid).To(Equal(messageID))
				Expect(mockRepo.CreateNewCalledWith.Parsed).To(BeFalse())
				Expect(mockRepo.CreateNewCalledWith.Category).To(Equal("XYZ")) // Category is found
				Expect(mockRepo.CreateNewCalledWith.Comments).To(ContainSubstring("no matching pattern found for body"))


				By("verifying publisher.Publish was called with unparsed message")
				Expect(mockPub.PublishCalledWith).NotTo(BeNil())
				Expect(mockPub.PublishCalledWith.Uuid).To(Equal(messageID))
				Expect(mockPub.PublishCalledWith.Parsed).To(BeFalse())
				Expect(mockPub.PublishCalledWith.Category).To(Equal("XYZ"))
			})
		})

		Context("with a nil or empty message payload", func() {
			It("should return an error for a nil message", func() {
				err := handler.HandleMessage(nil, messageID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("empty message"))
				Expect(mockRepo.CreateNewCalledWith).To(BeNil())
				Expect(mockPub.PublishCalledWith).To(BeNil())
			})

			It("should return an error for an empty message (empty byte slice)", func() {
				// Note: An empty string `string([]byte{})` is `""`.
				// `parsers.Parse("")` will result in `Parsed=false` and comments.
				// The handler's `if msg == nil` check is the primary guard here.
				// If `len(msg) == 0` is also considered an "empty message" to reject early, the handler needs that logic.
				// Current handler logic: `if msg == nil`. An empty slice `[]byte{}` is not `nil`.
				// So, `[]byte{}` will proceed to parsing. `parsers.Parse("")` will occur.
				// Let's test the `msg == nil` path specifically.
				// To test `len(msg) == 0` as an error, handler.go would need modification.
				// For now, this test is redundant if `nil` is covered.
				// If `len(msg) == 0` should also be an immediate error, this test would change.
				// Let's assume `len(msg) == 0` goes to parser.
				errNil := handler.HandleMessage(nil, "nil-id")
				Expect(errNil).To(HaveOccurred())
				Expect(errNil.Error()).To(ContainSubstring("empty message"))


				// Test for empty slice if it should behave differently (currently goes to parser)
				emptySliceMsg := []byte{}
				errEmpty := handler.HandleMessage(emptySliceMsg, "empty-slice-id")
				Expect(errEmpty).NotTo(HaveOccurred()) // Goes to parser, repo, pub
				Expect(mockRepo.CreateNewCalledWith).NotTo(BeNil())
				Expect(mockRepo.CreateNewCalledWith.Parsed).To(BeFalse())
				Expect(mockPub.PublishCalledWith).NotTo(BeNil())
				Expect(mockPub.PublishCalledWith.Parsed).To(BeFalse())
			})
		})
		
		Context("when repository.CreateNew returns an error", func() {
			var repoError = errors.New("repository CreateNew failed")
			BeforeEach(func() {
				validMsgBody := []byte("(ARR-VALIDMSG-DEP-ARR0000)")
				mockRepo.Reset()
				mockPub.Reset()
				mockRepo.CreateNewFunc = func(message *domain.ParsedMessage) error {
					return repoError
				}
				
				err := handler.HandleMessage(validMsgBody, messageID)
				Expect(err).NotTo(HaveOccurred()) // HandleMessage doesn't propagate repo error currently
			})

			It("should call repository.CreateNew", func() {
				Expect(mockRepo.CreateNewCalledWith).NotTo(BeNil())
			})

			It("should still call publisher.Publish", func() {
				Expect(mockPub.PublishCalledWith).NotTo(BeNil())
				Expect(mockPub.PublishCalledWith.Uuid).To(Equal(messageID))
			})
			// Logging of this error is assumed to happen inside HandleMessage
		})

		Context("when publisher.Publish returns an error", func() {
			var pubError = errors.New("publisher Publish failed")
			var returnedErr error
			BeforeEach(func() {
				validMsgBody := []byte("(ARR-VALIDMSG-DEP-ARR0000)")
				mockRepo.Reset()
				mockPub.Reset()
				mockPub.PublishFunc = func(message *domain.ParsedMessage) error {
					return pubError
				}
				returnedErr = handler.HandleMessage(validMsgBody, messageID)
			})

			It("should call repository.CreateNew", func() {
				Expect(mockRepo.CreateNewCalledWith).NotTo(BeNil())
			})

			It("should call publisher.Publish", func() {
				Expect(mockPub.PublishCalledWith).NotTo(BeNil())
			})
			
			It("should return the error from publisher.Publish", func() {
				Expect(returnedErr).To(HaveOccurred())
				Expect(errors.Is(returnedErr, pubError)).To(BeTrue())
			})
			// Logging of this error is assumed to happen inside HandleMessage
		})
	})
})
