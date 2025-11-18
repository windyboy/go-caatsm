package nats

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("isDevLikeEnv", func() {
		BeforeEach(func() {
			// Save original value
			originalEnv := os.Getenv("GO_ENV")
			DeferCleanup(func() {
				if originalEnv == "" {
					if err := os.Unsetenv("GO_ENV"); err != nil {
						// Environment variables are optional, ignore cleanup errors in tests
						_ = err
					}
				} else {
					if err := os.Setenv("GO_ENV", originalEnv); err != nil {
						// Environment variables are optional, ignore cleanup errors in tests
						_ = err
					}
				}
			})
		})

		It("returns true for dev environment", func() {
			if err := os.Setenv("GO_ENV", "dev"); err != nil {
				// Environment variables are optional, ignore setup errors in tests
				_ = err
			}
			Expect(isDevLikeEnv()).To(BeTrue())
		})

		It("returns true for development environment", func() {
			if err := os.Setenv("GO_ENV", "development"); err != nil {
				// Environment variables are optional, ignore setup errors in tests
				_ = err
			}
			Expect(isDevLikeEnv()).To(BeTrue())
		})

		It("returns true for test environment", func() {
			if err := os.Setenv("GO_ENV", "test"); err != nil {
				// Environment variables are optional, ignore setup errors in tests
				_ = err
			}
			Expect(isDevLikeEnv()).To(BeTrue())
		})

		It("returns true for testing environment", func() {
			if err := os.Setenv("GO_ENV", "testing"); err != nil {
				// Environment variables are optional, ignore setup errors in tests
				_ = err
			}
			Expect(isDevLikeEnv()).To(BeTrue())
		})

		It("returns true for empty environment", func() {
			if err := os.Unsetenv("GO_ENV"); err != nil {
				// Environment variables are optional, ignore cleanup errors in tests
				_ = err
			}
			Expect(isDevLikeEnv()).To(BeTrue())
		})

		It("returns false for production environment", func() {
			if err := os.Setenv("GO_ENV", "prod"); err != nil {
				// Environment variables are optional, ignore setup errors in tests
				_ = err
			}
			Expect(isDevLikeEnv()).To(BeFalse())
		})

		It("returns false for production environment (uppercase)", func() {
			if err := os.Setenv("GO_ENV", "PROD"); err != nil {
				// Environment variables are optional, ignore setup errors in tests
				_ = err
			}
			Expect(isDevLikeEnv()).To(BeFalse())
		})
	})

	Describe("sanitizeURLForLogging", func() {
		It("removes credentials from URLs", func() {
			url := "nats://user:pass@localhost:4222"
			Expect(sanitizeURLForLogging(url)).To(Equal("nats://localhost:4222"))
		})

		It("handles URLs without credentials", func() {
			url := "nats://localhost:4222"
			Expect(sanitizeURLForLogging(url)).To(Equal("nats://localhost:4222"))
		})

		It("handles empty strings", func() {
			Expect(sanitizeURLForLogging("")).To(Equal(""))
		})

		It("handles invalid URLs", func() {
			url := "://invalid"
			result := sanitizeURLForLogging(url)
			Expect(result).To(Equal("***"))
		})

		It("handles URLs with user but no password", func() {
			url := "nats://user@localhost:4222"
			Expect(sanitizeURLForLogging(url)).To(Equal("nats://localhost:4222"))
		})
	})
})
