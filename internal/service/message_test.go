package service

import (
	"context"
	"errors"
	"time"

	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/internal/iface"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeRepository struct {
	createFunc func(ctx context.Context, message *domain.ParsedMessage) error
}

func (f *fakeRepository) Create(ctx context.Context, message *domain.ParsedMessage) error {
	if f.createFunc != nil {
		return f.createFunc(ctx, message)
	}
	return nil
}

func (f *fakeRepository) CreateBatch(ctx context.Context, messages []*domain.ParsedMessage) error {
	return nil
}

func (f *fakeRepository) GetByID(ctx context.Context, id string) (*domain.ParsedMessage, error) {
	return nil, nil
}

func (f *fakeRepository) List(ctx context.Context, filters *iface.TelegramFilters) ([]*domain.ParsedMessage, int64, error) {
	return nil, 0, nil
}

func (f *fakeRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.ParsedMessage, error) {
	return nil, nil
}

func (f *fakeRepository) Count(ctx context.Context, filters *iface.TelegramFilters) (int64, error) {
	return 0, nil
}

type fakePublisher struct {
	publishFunc func(ctx context.Context, subject string, message interface{}) error
}

func (f *fakePublisher) Publish(ctx context.Context, subject string, message interface{}) error {
	if f.publishFunc != nil {
		return f.publishFunc(ctx, subject, message)
	}
	return nil
}

func (f *fakePublisher) PublishAsync(ctx context.Context, subject string, message interface{}) error {
	return nil
}

var _ = Describe("MessageService", func() {
	var (
		svc       *MessageService
		cfg       *config.Config
		repo      *fakeRepository
		pub       *fakePublisher
		ctx       context.Context
		validMsg  []byte
		messageID string
	)

	BeforeEach(func() {
		cfg = &config.Config{
			Publisher: config.PublisherConfig{Topic: "test.topic"},
		}
		repo = &fakeRepository{}
		pub = &fakePublisher{}
		svc = NewMessageService(cfg, repo, pub)
		ctx = context.Background()
		validMsg = []byte("ZCZC TMQ2530 141614\nGG ZBTJZXZX\n141614 ZSHCZTZX\n(ARR-CES5470-ZBTJ-ZSHC1614)\nNNNN")
		messageID = "test-message-id"
	})

	Context("validation", func() {
		It("returns an error for empty messages", func() {
			err := svc.ProcessMessage(ctx, []byte{}, messageID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty message"))
		})

		It("returns an error for oversized messages", func() {
			largeMsg := make([]byte, MaxMessageSize+1)
			err := svc.ProcessMessage(ctx, largeMsg, messageID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
		})
	})

	Context("persistence", func() {
		It("propagates repository errors", func() {
			repoErr := errors.New("repo failure")
			repo.createFunc = func(ctx context.Context, message *domain.ParsedMessage) error {
				return repoErr
			}

			err := svc.ProcessMessage(ctx, validMsg, messageID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("repository error"))
		})
	})

	Context("publishing", func() {
		It("propagates publisher errors", func() {
			pubErr := errors.New("publish failure")
			pub.publishFunc = func(ctx context.Context, subject string, message interface{}) error {
				Expect(subject).To(Equal(cfg.Publisher.Topic))
				return pubErr
			}

			err := svc.ProcessMessage(ctx, validMsg, messageID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("publish error"))
		})
	})

	Context("success path", func() {
		It("stores and publishes messages", func() {
			repoCalled := false
			pubCalled := false

			repo.createFunc = func(ctx context.Context, message *domain.ParsedMessage) error {
				repoCalled = true
				Expect(message).NotTo(BeNil())
				return nil
			}
			pub.publishFunc = func(ctx context.Context, subject string, message interface{}) error {
				pubCalled = true
				Expect(subject).To(Equal(cfg.Publisher.Topic))
				return nil
			}

			err := svc.ProcessMessage(ctx, validMsg, messageID)
			Expect(err).NotTo(HaveOccurred())
			Expect(repoCalled).To(BeTrue())
			Expect(pubCalled).To(BeTrue())
		})
	})
})
