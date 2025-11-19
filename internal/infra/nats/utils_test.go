package nats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {

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
