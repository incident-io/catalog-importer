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

var _ = Describe("Output CompileExpressions", func() {
	var o output.Output

	BeforeEach(func() {
		o = output.Output{
			Source: output.SourceConfig{
				Filter:     null.StringFrom("$.kind == 'Component'"),
				Name:       "$.metadata.name",
				ExternalID: "$.metadata.uid",
				Rank:       null.StringFrom("$.rank"),
				Aliases:    []string{"$.metadata.title"},
			},
			Attributes: []*output.Attribute{
				{ID: "owner", Source: null.StringFrom("$.spec.owner.replace('group:', '')")},
				{ID: "tier"},
			},
		}
	})

	It("accepts valid expressions", func() {
		Expect(o.CompileExpressions()).To(BeEmpty())
	})

	It("reports each invalid expression with where it is configured", func() {
		o.Source.Filter = null.StringFrom("$.kind ==")
		o.Source.Aliases = []string{"$.metadata.title", "$.metadata.(name"}
		o.Attributes[0].Source = null.StringFrom("$.spec.owner.replace('group:', ''")

		errs := o.CompileExpressions()
		Expect(errs).To(HaveLen(3))
		Expect(errs[0].Error()).To(HavePrefix("source.filter: "))
		Expect(errs[1].Error()).To(HavePrefix("source.aliases.1: "))
		Expect(errs[2].Error()).To(HavePrefix("attributes.owner: "))
	})

	// Attribute IDs created in the dashboard are ULIDs, which start with a digit, so the
	// default source expression ($.01KG...) is not valid Javascript.
	const ulid = "01KG2XN4AG7HGJCFXF305GH05M"

	It("skips backlink attributes, which have no source expression", func() {
		o.Attributes = append(o.Attributes, &output.Attribute{
			ID:                ulid,
			BacklinkAttribute: null.StringFrom("01KG2TVW9MTB6V5EP56G6DGR30"),
		})
		Expect(o.CompileExpressions()).To(BeEmpty())
	})

	It("skips path attributes, which have no source expression", func() {
		o.Attributes = append(o.Attributes, &output.Attribute{
			ID:   ulid,
			Path: []string{"01KG2TVW9MTB6V5EP56G6DGR30", "01KG2TVW9MTB6V5EP56G6DGR31"},
		})
		Expect(o.CompileExpressions()).To(BeEmpty())
	})

	It("skips schema-only attributes, which have no source expression", func() {
		o.Attributes = append(o.Attributes, &output.Attribute{ID: ulid, SchemaOnly: true})
		Expect(o.CompileExpressions()).To(BeEmpty())
	})

	It("reports a synced attribute whose default source expression is invalid", func() {
		o.Attributes = append(o.Attributes, &output.Attribute{ID: ulid})

		errs := o.CompileExpressions()
		Expect(errs).To(HaveLen(1))
		Expect(errs[0].Error()).To(HavePrefix("attributes." + ulid + ": "))
	})
})
