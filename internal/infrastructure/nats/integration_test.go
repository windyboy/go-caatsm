package nats

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/internal/iface"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type integrationMockRepository struct {
	Messages []*domain.ParsedMessage
	Errors   []error
}

func (m *integrationMockRepository) Create(ctx context.Context, message *domain.ParsedMessage) error {
	if len(m.Errors) > 0 {
		err := m.Errors[0]
		m.Errors = m.Errors[1:]
		return err
	}
	m.Messages = append(m.Messages, message)
	return nil
}

func (m *integrationMockRepository) CreateBatch(ctx context.Context, messages []*domain.ParsedMessage) error {
	return nil
}

func (m *integrationMockRepository) GetByID(ctx context.Context, id string) (*domain.ParsedMessage, error) {
	return nil, nil
}

func (m *integrationMockRepository) List(ctx context.Context, filters *iface.TelegramFilters) ([]*domain.ParsedMessage, int64, error) {
	return nil, 0, nil
}

func (m *integrationMockRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.ParsedMessage, error) {
	return nil, nil
}

func (m *integrationMockRepository) Count(ctx context.Context, filters *iface.TelegramFilters) (int64, error) {
	return 0, nil
}

type integrationMockPublisher struct {
	Messages        []interface{}
	PublishErrors   []error
	PublishAsyncErr []error
}

func (m *integrationMockPublisher) Publish(ctx context.Context, subject string, message interface{}) error {
	if len(m.PublishErrors) > 0 {
		err := m.PublishErrors[0]
		m.PublishErrors = m.PublishErrors[1:]
		return err
	}
	m.Messages = append(m.Messages, message)
	return nil
}

func (m *integrationMockPublisher) PublishAsync(ctx context.Context, subject string, message interface{}) error {
	if len(m.PublishAsyncErr) > 0 {
		err := m.PublishAsyncErr[0]
		m.PublishAsyncErr = m.PublishAsyncErr[1:]
		return err
	}
	m.Messages = append(m.Messages, message)
	return nil
}

var _ = Describe("Integration Tests", func() {
	Describe("Message Flow", func() {
		Context("End-to-end message processing", func() {
			It("successfully processes a valid message", func() {
				cfg := &config.Config{
					Publisher: config.PublisherConfig{Topic: "test.topic"},
				}
				mockRepo := &integrationMockRepository{
					Messages: make([]*domain.ParsedMessage, 0),
				}
				mockPub := &integrationMockPublisher{
					Messages: make([]interface{}, 0),
				}
				handler := NewHandler(cfg, mockPub, mockRepo)

				validMsg := []byte("ZCZC TMQ2530 141614\nGG ZBTJZXZX\n141614 ZSHCZTZX\n(ARR-CES5470-ZBTJ-ZSHC1614)\nNNNN")
				messageID := "test-integration-id"

				ctx := context.Background()
				err := handler.HandleMessage(ctx, validMsg, messageID)

				Expect(err).NotTo(HaveOccurred())
				Expect(len(mockRepo.Messages)).To(BeNumerically(">", 0))
			})
		})

		Context("Concurrent message processing", func() {
			It("handles multiple messages", func() {
				cfg := &config.Config{
					Publisher: config.PublisherConfig{Topic: "test.topic"},
				}
				mockRepo := &integrationMockRepository{
					Messages: make([]*domain.ParsedMessage, 0),
				}
				mockPub := &integrationMockPublisher{
					Messages: make([]interface{}, 0),
				}
				handler := NewHandler(cfg, mockPub, mockRepo)

				validMsg := []byte("ZCZC TMQ2530 141614\nGG ZBTJZXZX\n141614 ZSHCZTZX\n(ARR-CES5470-ZBTJ-ZSHC1614)\nNNNN")
				ctx := context.Background()

				done := make(chan bool, 10)
				for i := 0; i < 10; i++ {
					go func(id int) {
						err := handler.HandleMessage(ctx, validMsg, fmt.Sprintf("msg-%d", id))
						Expect(err).NotTo(HaveOccurred())
						done <- true
					}(i)
				}

				for i := 0; i < 10; i++ {
					<-done
				}

				Expect(len(mockRepo.Messages)).To(Equal(10))
			})
		})
	})

	Describe("Graceful Shutdown", func() {
		Context("when context is cancelled", func() {
			It("stops processing", func() {
				ctx, cancel := context.WithCancel(context.Background())

				go func() {
					time.Sleep(100 * time.Millisecond)
					cancel()
				}()

				<-ctx.Done()
				Expect(ctx.Err()).To(Equal(context.Canceled))
			})
		})

		Context("when receiving SIGTERM signal", func() {
			It("handles signal gracefully", func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGTERM)

				go func() {
					time.Sleep(50 * time.Millisecond)
					sigChan <- syscall.SIGTERM
				}()

				sig := <-sigChan
				Expect(sig).To(Equal(syscall.SIGTERM))
			})
		})
	})
})
