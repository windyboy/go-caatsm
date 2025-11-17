package postgres

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgres(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Postgres Suite")
}

var _ = Describe("sanitizePostgresURLForLogging", func() {
	It("returns a masked URL when config is empty", func() {
		Expect(sanitizePostgresURLForLogging(nil)).To(Equal("postgres://***@***/***"))
	})

	It("returns a sanitized URL when fields are populated", func() {
		cfg, err := pgxpool.ParseConfig("postgres://user:secret@db.local:6543/telegrams")
		Expect(err).NotTo(HaveOccurred())
		Expect(sanitizePostgresURLForLogging(cfg)).To(Equal("postgres://***@db.local:6543/telegrams"))
	})
})
