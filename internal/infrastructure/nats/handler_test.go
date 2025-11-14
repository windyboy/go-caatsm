package nats

import (
	"context"
	"errors"
	"time"

	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/internal/iface"
	"caatsm/internal/service"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockRepository struct {
	CreateFunc func(ctx context.Context, message *domain.ParsedMessage) error
}

func (m *mockRepository) Create(ctx context.Context, message *domain.ParsedMessage) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, message)
	}
	return nil
}

func (m *mockRepository) CreateBatch(ctx context.Context, messages []*domain.ParsedMessage) error {
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*domain.ParsedMessage, error) {
	return nil, nil
}

func (m *mockRepository) List(ctx context.Context, filters *iface.TelegramFilters) ([]*domain.ParsedMessage, int64, error) {
	return nil, 0, nil
}

func (m *mockRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.ParsedMessage, error) {
	return nil, nil
}

func (m *mockRepository) Count(ctx context.Context, filters *iface.TelegramFilters) (int64, error) {
	return 0, nil
}

type mockPublisher struct {
	PublishFunc      func(ctx context.Context, subject string, message interface{}) error
	PublishAsyncFunc func(ctx context.Context, subject string, message interface{}) error
}

func (m *mockPublisher) Publish(ctx context.Context, subject string, message interface{}) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, subject, message)
	}
	return nil
}

func (m *mockPublisher) PublishAsync(ctx context.Context, subject string, message interface{}) error {
	if m.PublishAsyncFunc != nil {
		return m.PublishAsyncFunc(ctx, subject, message)
	}
	return nil
}

var _ = Describe("MessageHandler", func() {
	var (
		handler   *MessageHandler
		mockRepo  *mockRepository
		mockPub   *mockPublisher
		cfg       *config.Config
		ctx       context.Context
		validMsg  []byte
		messageID string
	)

	BeforeEach(func() {
		ctx = context.Background()
		cfg = &config.Config{
			Publisher: config.PublisherConfig{Topic: "test.topic"},
		}
		mockRepo = &mockRepository{}
		mockPub = &mockPublisher{}
		handler = NewHandler(cfg, mockPub, mockRepo)
		validMsg = []byte("ZCZC TMQ2530 141614\nGG ZBTJZXZX\n141614 ZSHCZTZX\n(ARR-CES5470-ZBTJ-ZSHC1614)\nNNNN")
		messageID = "test-message-id"
	})

	Describe("HandleMessage", func() {
		Context("when message is empty", func() {
			It("returns an error for nil message", func() {
				err := handler.HandleMessage(ctx, nil, messageID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("empty message"))
			})

			It("returns an error for empty message", func() {
				err := handler.HandleMessage(ctx, []byte{}, messageID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("empty message"))
			})
		})

		Context("when message is too large", func() {
			It("returns an error", func() {
				largeMsg := make([]byte, service.MaxMessageSize+1)
				err := handler.HandleMessage(ctx, largeMsg, messageID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
			})
		})

		Context("when repository returns an error", func() {
			It("propagates the error", func() {
				repoError := errors.New("repository error")
				mockRepo.CreateFunc = func(ctx context.Context, message *domain.ParsedMessage) error {
					return repoError
				}

				err := handler.HandleMessage(ctx, validMsg, messageID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("repository error"))
			})
		})

		Context("when publisher returns an error", func() {
			It("propagates the error", func() {
				pubError := errors.New("publisher error")
				mockRepo.CreateFunc = func(ctx context.Context, message *domain.ParsedMessage) error {
					return nil
				}
				mockPub.PublishFunc = func(ctx context.Context, subject string, message interface{}) error {
					Expect(subject).To(Equal(cfg.Publisher.Topic))
					return pubError
				}

				err := handler.HandleMessage(ctx, validMsg, messageID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("publish error"))
			})
		})

		Context("when message is valid", func() {
			It("stores and publishes the message", func() {
				repoCalled := false
				pubCalled := false

				mockRepo.CreateFunc = func(ctx context.Context, message *domain.ParsedMessage) error {
					repoCalled = true
					Expect(message).NotTo(BeNil())
					return nil
				}
				mockPub.PublishFunc = func(ctx context.Context, subject string, message interface{}) error {
					pubCalled = true
					Expect(subject).To(Equal(cfg.Publisher.Topic))
					Expect(message).NotTo(BeNil())
					return nil
				}

				err := handler.HandleMessage(ctx, validMsg, messageID)
				Expect(err).NotTo(HaveOccurred())
				Expect(repoCalled).To(BeTrue())
				Expect(pubCalled).To(BeTrue())
			})
		})
	})
})
