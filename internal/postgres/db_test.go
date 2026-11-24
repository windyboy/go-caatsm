package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sanitizePostgresURLForLogging", func() {
	It("returns a masked URL when config is nil", func() {
		Expect(sanitizePostgresURLForLogging(nil)).To(Equal("postgres://***@***/***"))
	})

	It("strips credentials while keeping host, port, and database", func() {
		cfg, err := pgxpool.ParseConfig("postgres://user:secret@db.local:6543/telegrams")
		Expect(err).NotTo(HaveOccurred())
		Expect(sanitizePostgresURLForLogging(cfg)).To(Equal("postgres://***@db.local:6543/telegrams"))
	})
})
