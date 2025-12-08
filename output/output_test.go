package output_test

import (
	"github.com/incident-io/catalog-importer/v2/output"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/guregu/null.v3"
)

var _ = Describe("Output validation", func() {
	Describe("TypeName validation", func() {
		It("accepts type names with numbers", func() {
			o := output.Output{
				Name:        "Service 01",
				Description: "A service",
				TypeName:    `Custom["Service01"]`,
				Source: output.SourceConfig{
					Name:       "$.name",
					ExternalID: "$.id",
				},
			}
			Expect(o.Validate()).To(Succeed())
		})

		It("accepts type names with lowercase first letter", func() {
			o := output.Output{
				Name:        "My Service",
				Description: "A service",
				TypeName:    `Custom["service"]`,
				Source: output.SourceConfig{
					Name:       "$.name",
					ExternalID: "$.id",
				},
			}
			Expect(o.Validate()).To(Succeed())
		})

		It("accepts type names with mixed alphanumeric characters", func() {
			o := output.Output{
				Name:        "Service",
				Description: "A service",
				TypeName:    `Custom["Service123abc"]`,
				Source: output.SourceConfig{
					Name:       "$.name",
					ExternalID: "$.id",
				},
			}
			Expect(o.Validate()).To(Succeed())
		})

		It("rejects type names with special characters", func() {
			o := output.Output{
				Name:        "Service",
				Description: "A service",
				TypeName:    `Custom["Service-01"]`,
				Source: output.SourceConfig{
					Name:       "$.name",
					ExternalID: "$.id",
				},
			}
			Expect(o.Validate()).To(HaveOccurred())
		})

		It("rejects type names with spaces", func() {
			o := output.Output{
				Name:        "Service",
				Description: "A service",
				TypeName:    `Custom["Service 01"]`,
				Source: output.SourceConfig{
					Name:       "$.name",
					ExternalID: "$.id",
				},
			}
			Expect(o.Validate()).To(HaveOccurred())
		})

		It("rejects type names without Custom prefix", func() {
			o := output.Output{
				Name:        "Service",
				Description: "A service",
				TypeName:    "Service01",
				Source: output.SourceConfig{
					Name:       "$.name",
					ExternalID: "$.id",
				},
			}
			Expect(o.Validate()).To(HaveOccurred())
		})
	})

	Describe("Required fields", func() {
		It("requires name", func() {
			o := output.Output{
				Description: "A service",
				TypeName:    `Custom["Service"]`,
				Source: output.SourceConfig{
					Name:       "$.name",
					ExternalID: "$.id",
				},
			}
			Expect(o.Validate()).To(HaveOccurred())
		})

		It("requires description", func() {
			o := output.Output{
				Name:     "Service",
				TypeName: `Custom["Service"]`,
				Source: output.SourceConfig{
					Name:       "$.name",
					ExternalID: "$.id",
				},
			}
			Expect(o.Validate()).To(HaveOccurred())
		})

		It("requires type_name", func() {
			o := output.Output{
				Name:        "Service",
				Description: "A service",
				Source: output.SourceConfig{
					Name:       "$.name",
					ExternalID: "$.id",
				},
			}
			Expect(o.Validate()).To(HaveOccurred())
		})
	})
})

var _ = Describe("Attribute validation", func() {
	It("requires either type or enum", func() {
		attr := output.Attribute{
			ID:   "test",
			Name: "Test",
		}
		Expect(attr.Validate()).To(HaveOccurred())
	})

	It("accepts type without enum", func() {
		attr := output.Attribute{
			ID:   "test",
			Name: "Test",
			Type: null.StringFrom("String"),
		}
		Expect(attr.Validate()).To(Succeed())
	})

	It("accepts enum without type", func() {
		attr := output.Attribute{
			ID:   "test",
			Name: "Test",
			Enum: &output.AttributeEnum{
				Name:        "TestEnum",
				Description: "An enum",
				TypeName:    `Custom["TestEnum"]`,
			},
		}
		Expect(attr.Validate()).To(Succeed())
	})

	It("rejects both type and enum", func() {
		attr := output.Attribute{
			ID:   "test",
			Name: "Test",
			Type: null.StringFrom("String"),
			Enum: &output.AttributeEnum{
				Name:        "TestEnum",
				Description: "An enum",
				TypeName:    `Custom["TestEnum"]`,
			},
		}
		Expect(attr.Validate()).To(HaveOccurred())
	})
})
