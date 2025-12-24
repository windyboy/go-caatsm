package validator_test

import (
	"caatsm/internal/adapter/dto"
	"caatsm/internal/adapter/validator"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AFTN Validator", func() {
	Describe("ValidatePriorityIndicator", func() {
		Context("with valid priority indicators", func() {
			It("accepts FF (Flash)", func() {
				err := validator.ValidatePriorityIndicator("FF")
				Expect(err).To(BeNil())
			})

			It("accepts GG (Immediate)", func() {
				err := validator.ValidatePriorityIndicator("GG")
				Expect(err).To(BeNil())
			})

			It("accepts QU (Distress)", func() {
				err := validator.ValidatePriorityIndicator("QU")
				Expect(err).To(BeNil())
			})

			It("accepts DD (Delay)", func() {
				err := validator.ValidatePriorityIndicator("DD")
				Expect(err).To(BeNil())
			})

			It("accepts SS (Service)", func() {
				err := validator.ValidatePriorityIndicator("SS")
				Expect(err).To(BeNil())
			})

			It("accepts KK (Correction)", func() {
				err := validator.ValidatePriorityIndicator("KK")
				Expect(err).To(BeNil())
			})

			It("accepts lowercase with trimming", func() {
				err := validator.ValidatePriorityIndicator("  ff  ")
				Expect(err).To(BeNil())
			})

			It("accepts empty string (optional field)", func() {
				err := validator.ValidatePriorityIndicator("")
				Expect(err).To(BeNil())
			})
		})

		Context("with invalid priority indicators", func() {
			It("rejects invalid code XX", func() {
				err := validator.ValidatePriorityIndicator("XX")
				Expect(err).ToNot(BeNil())
				Expect(validator.IsAFTNError(err)).To(BeTrue())
				Expect(validator.GetAFTNErrorType(err)).To(Equal("priority_indicator"))
			})

			It("rejects single character", func() {
				err := validator.ValidatePriorityIndicator("F")
				Expect(err).ToNot(BeNil())
			})

			It("rejects three characters", func() {
				err := validator.ValidatePriorityIndicator("FFF")
				Expect(err).ToNot(BeNil())
			})
		})
	})

	Describe("ValidateICAOAddress", func() {
		Context("with valid ICAO addresses", func() {
			It("accepts ZBTJ (Beijing)", func() {
				err := validator.ValidateICAOAddress("ZBTJ")
				Expect(err).To(BeNil())
			})

			It("accepts KLAX (Los Angeles)", func() {
				err := validator.ValidateICAOAddress("KLAX")
				Expect(err).To(BeNil())
			})

			It("accepts ZGGG (Guangzhou)", func() {
				err := validator.ValidateICAOAddress("ZGGG")
				Expect(err).To(BeNil())
			})

			It("accepts alphanumeric codes like Z999", func() {
				err := validator.ValidateICAOAddress("Z999")
				Expect(err).To(BeNil())
			})

			It("accepts 1ABC", func() {
				err := validator.ValidateICAOAddress("1ABC")
				Expect(err).To(BeNil())
			})

			It("accepts lowercase with trimming", func() {
				err := validator.ValidateICAOAddress("  zbtj  ")
				Expect(err).To(BeNil())
			})

			It("accepts empty string (optional field)", func() {
				err := validator.ValidateICAOAddress("")
				Expect(err).To(BeNil())
			})
		})

		Context("with invalid ICAO addresses", func() {
			It("rejects too short (3 chars)", func() {
				err := validator.ValidateICAOAddress("ZBT")
				Expect(err).ToNot(BeNil())
				Expect(validator.IsAFTNError(err)).To(BeTrue())
				Expect(validator.GetAFTNErrorType(err)).To(Equal("icao_address"))
			})

			It("rejects too long (5 chars)", func() {
				err := validator.ValidateICAOAddress("ZBTJX")
				Expect(err).ToNot(BeNil())
			})

			It("rejects special characters", func() {
				err := validator.ValidateICAOAddress("ZB-J")
				Expect(err).ToNot(BeNil())
			})

			It("rejects spaces", func() {
				err := validator.ValidateICAOAddress("ZB J")
				Expect(err).ToNot(BeNil())
			})
		})
	})

	Describe("ValidateDateTime", func() {
		Context("with valid datetime values", func() {
			It("accepts 151430 (15th day, 14:30)", func() {
				err := validator.ValidateDateTime("151430")
				Expect(err).To(BeNil())
			})

			It("accepts 010000 (1st day, 00:00)", func() {
				err := validator.ValidateDateTime("010000")
				Expect(err).To(BeNil())
			})

			It("accepts 312359 (31st day, 23:59)", func() {
				err := validator.ValidateDateTime("312359")
				Expect(err).To(BeNil())
			})

			It("accepts empty string (optional field)", func() {
				err := validator.ValidateDateTime("")
				Expect(err).To(BeNil())
			})
		})

		Context("with invalid datetime values", func() {
			It("rejects non-numeric", func() {
				err := validator.ValidateDateTime("15A430")
				Expect(err).ToNot(BeNil())
				Expect(validator.IsAFTNError(err)).To(BeTrue())
				Expect(validator.GetAFTNErrorType(err)).To(Equal("datetime"))
			})

			It("rejects too short (5 digits)", func() {
				err := validator.ValidateDateTime("15143")
				Expect(err).ToNot(BeNil())
			})

			It("rejects too long (7 digits)", func() {
				err := validator.ValidateDateTime("1514301")
				Expect(err).ToNot(BeNil())
			})

			It("rejects invalid day (00)", func() {
				err := validator.ValidateDateTime("001430")
				Expect(err).ToNot(BeNil())
			})

			It("rejects invalid day (32)", func() {
				err := validator.ValidateDateTime("321430")
				Expect(err).ToNot(BeNil())
			})

			It("rejects invalid hour (24)", func() {
				err := validator.ValidateDateTime("152430")
				Expect(err).ToNot(BeNil())
			})

			It("rejects invalid minute (60)", func() {
				err := validator.ValidateDateTime("151460")
				Expect(err).ToNot(BeNil())
			})

			It("rejects invalid minute (99)", func() {
				err := validator.ValidateDateTime("151499")
				Expect(err).ToNot(BeNil())
			})
		})
	})

	Describe("ValidateTelegram", func() {
		Context("with valid telegram", func() {
			It("accepts telegram with all valid fields", func() {
				telegram := &dto.ParsedTelegram{
					PriorityIndicator:  "FF",
					PrimaryAddress:     "ZBTJ",
					Originator:         "KLAX",
					DateTime:           "151430",
					OriginatorDateTime: "151425",
				}
				err := validator.ValidateTelegram(telegram)
				Expect(err).To(BeNil())
			})

			It("accepts telegram with empty optional fields", func() {
				telegram := &dto.ParsedTelegram{
					PriorityIndicator:  "",
					PrimaryAddress:     "ZBTJ",
					Originator:         "",
					DateTime:           "151430",
					OriginatorDateTime: "",
				}
				err := validator.ValidateTelegram(telegram)
				Expect(err).To(BeNil())
			})

			It("accepts nil telegram", func() {
				err := validator.ValidateTelegram(nil)
				Expect(err).To(BeNil())
			})
		})

		Context("with invalid telegram fields", func() {
			It("reports invalid priority indicator", func() {
				telegram := &dto.ParsedTelegram{
					PriorityIndicator: "XX",
					PrimaryAddress:    "ZBTJ",
					DateTime:          "151430",
				}
				err := validator.ValidateTelegram(telegram)
				Expect(err).ToNot(BeNil())
				Expect(validator.IsAFTNError(err)).To(BeTrue())
			})

			It("reports invalid primary address", func() {
				telegram := &dto.ParsedTelegram{
					PriorityIndicator: "FF",
					PrimaryAddress:    "TOOLONG",
					DateTime:          "151430",
				}
				err := validator.ValidateTelegram(telegram)
				Expect(err).ToNot(BeNil())
				Expect(validator.IsAFTNError(err)).To(BeTrue())
			})

			It("reports invalid originator", func() {
				telegram := &dto.ParsedTelegram{
					PriorityIndicator: "FF",
					PrimaryAddress:    "ZBTJ",
					Originator:        "KL",
					DateTime:          "151430",
				}
				err := validator.ValidateTelegram(telegram)
				Expect(err).ToNot(BeNil())
				Expect(validator.IsAFTNError(err)).To(BeTrue())
			})

			It("reports invalid datetime", func() {
				telegram := &dto.ParsedTelegram{
					PriorityIndicator: "FF",
					PrimaryAddress:    "ZBTJ",
					DateTime:          "321430",
				}
				err := validator.ValidateTelegram(telegram)
				Expect(err).ToNot(BeNil())
				Expect(validator.IsAFTNError(err)).To(BeTrue())
			})

			It("reports multiple errors", func() {
				telegram := &dto.ParsedTelegram{
					PriorityIndicator:  "XX",
					PrimaryAddress:     "TOOLONG",
					Originator:         "KL",
					DateTime:           "321430",
					OriginatorDateTime: "991499",
				}
				err := validator.ValidateTelegram(telegram)
				Expect(err).ToNot(BeNil())
				Expect(validator.IsAFTNError(err)).To(BeTrue())
				Expect(validator.GetAFTNErrorType(err)).To(Equal("multiple_errors"))
				Expect(err.Error()).To(ContainSubstring("AFTN validation failed"))
			})
		})
	})

	Describe("IsAFTNError", func() {
		It("returns true for AFTNError", func() {
			err := validator.ValidatePriorityIndicator("XX")
			Expect(validator.IsAFTNError(err)).To(BeTrue())
		})

		It("returns true for AFTNValidationErrors", func() {
			telegram := &dto.ParsedTelegram{
				PriorityIndicator: "XX",
				PrimaryAddress:    "TOOLONG",
			}
			err := validator.ValidateTelegram(telegram)
			Expect(validator.IsAFTNError(err)).To(BeTrue())
		})

		It("returns false for nil error", func() {
			Expect(validator.IsAFTNError(nil)).To(BeFalse())
		})
	})

	Describe("GetAFTNErrorType", func() {
		It("extracts field name from AFTNError", func() {
			err := validator.ValidatePriorityIndicator("XX")
			Expect(validator.GetAFTNErrorType(err)).To(Equal("priority_indicator"))
		})

		It("returns 'multiple_errors' for AFTNValidationErrors", func() {
			telegram := &dto.ParsedTelegram{
				PriorityIndicator: "XX",
				PrimaryAddress:    "TOOLONG",
			}
			err := validator.ValidateTelegram(telegram)
			Expect(validator.GetAFTNErrorType(err)).To(Equal("multiple_errors"))
		})

		It("returns 'unknown' for non-AFTN errors", func() {
			errorType := validator.GetAFTNErrorType(nil)
			Expect(errorType).To(Equal("unknown"))
		})
	})
})
