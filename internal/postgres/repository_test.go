package postgres

import (
	"caatsm/internal/telegram"
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Repository guards", func() {
	It("validates nil messages and empty batches safely", func() {
		repo := NewRepository(nil, zap.NewNop())
		ctx := context.Background()

		Expect(repo.InsertOne(ctx, nil)).To(HaveOccurred())
		Expect(repo.InsertOne(ctx, &telegram.Message{})).To(HaveOccurred())
		Expect(repo.InsertBatch(ctx, nil)).To(Succeed())
		Expect(repo.InsertBatch(ctx, []*telegram.Message{})).To(Succeed())
		Expect(repo.InsertBatch(ctx, []*telegram.Message{{}})).To(HaveOccurred())
		Expect(repo.InsertRaw(ctx, nil)).To(HaveOccurred())
		Expect(repo.InsertRaw(ctx, &telegram.Message{})).To(HaveOccurred())
	})
})
