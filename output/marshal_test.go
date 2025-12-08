package output

import (
	"context"
	"os"

	kitlog "github.com/go-kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/source"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/samber/lo"
	"gopkg.in/guregu/null.v3"
)

var _ = Describe("Attribute", func() {
	Describe("IncludeInPayload", func() {
		It("includes normal attributes", func() {
			attr := Attribute{
				ID:   "normal_attr",
				Name: "Normal Attribute",
				Type: null.StringFrom("String"),
			}
			Expect(attr.IncludeInPayload()).To(BeTrue())
		})

		It("excludes schema-only attributes", func() {
			attr := Attribute{
				ID:         "schema_only_attr",
				Name:       "Schema Only Attribute",
				Type:       null.StringFrom("String"),
				SchemaOnly: true,
			}
			Expect(attr.IncludeInPayload()).To(BeFalse())
		})

		It("excludes backlink attributes", func() {
			attr := Attribute{
				ID:                "backlink_attr",
				Name:              "Backlink Attribute",
				Type:              null.StringFrom("OtherType"),
				BacklinkAttribute: null.StringFrom("source_attr"),
			}
			Expect(attr.IncludeInPayload()).To(BeFalse())
		})

		It("excludes path attributes", func() {
			attr := Attribute{
				ID:   "path_attr",
				Name: "Path Attribute",
				Type: null.StringFrom("String"),
				Path: []string{"step1", "step2"},
			}
			Expect(attr.IncludeInPayload()).To(BeFalse())
		})
	})
})

var _ = Describe("Marshalling data", func() {
	var (
		ctx               context.Context
		catalogTypeOutput *Output
		logger            kitlog.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()

		logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	})

	Describe("aliases", func() {
		BeforeEach(func() {
			sourceConfig := SourceConfig{
				Name:       "$.name",
				ExternalID: "$.external_id",
				Aliases:    []string{"$.aliases"},
			}

			catalogTypeOutput = &Output{
				Name:        "name",
				Description: "description",
				Source:      sourceConfig,
			}
		})

		When("Marshalling alias data where the entry has an array of aliases", func() {
			It("correctly populates the array on the resulting entry with all values", func() {
				sourceEntry := source.Entry{
					"id":          "P1234",
					"name":        "Component name 1",
					"description": "A super important component. A structurally integral component tbh.",
					"aliases":     []string{"aliasInAnArray", "anotherAliasInAnArray"},
				}

				entries := []source.Entry{sourceEntry}

				res, err := MarshalEntries(ctx, logger, catalogTypeOutput, entries)

				expectedAliasResult := []string{"aliasInAnArray", "anotherAliasInAnArray"}
				Expect(err).NotTo(HaveOccurred())
				Expect(res[0].Aliases).To(Equal(expectedAliasResult))

			})
		})

		When("Marshalling alias data where the entry has a single string alias", func() {
			It("correctly populates the alias array on the resulting entry with the single value", func() {
				sourceEntry := source.Entry{
					"id":          "P1235",
					"name":        "Component name 2",
					"description": "A super important component. A structurally integral component tbh.",
					"aliases":     "singleAlias",
				}
				entries := []source.Entry{sourceEntry}
				res, err := MarshalEntries(ctx, logger, catalogTypeOutput, entries)
				expectedAliasResult := []string{"singleAlias"}
				Expect(err).NotTo(HaveOccurred())
				Expect(res[0].Aliases).To(Equal(expectedAliasResult))
			})
		})
	})

	Describe("attributes", func() {
		BeforeEach(func() {
			sourceConfig := SourceConfig{
				Name:       "$.name",
				ExternalID: "$.external_id",
			}

			catalogTypeOutput = &Output{
				Name:        "name",
				Description: "description",
				Source:      sourceConfig,
				Attributes:  []*Attribute{{ID: "an_attribute", Name: "An attribute", Type: null.StringFrom("String"), Source: null.StringFrom("$.an_attribute")}},
			}
		})

		When("marshalling attributes that aren't present in the source entry", func() {
			It("doesn't include the attribute in the resulting entry", func() {
				sourceEntry := source.Entry{
					"id":          "P1235",
					"name":        "Component name 2",
					"description": "A super important component. A structurally integral component tbh.",
				}
				entries := []source.Entry{sourceEntry}
				res, err := MarshalEntries(ctx, logger, catalogTypeOutput, entries)
				Expect(err).NotTo(HaveOccurred())
				Expect(res[0].AttributeValues).To(BeEmpty())
			})
		})
	})
})

var _ = Describe("MarshalType", func() {
	var (
		output *Output
	)

	BeforeEach(func() {
		output = &Output{
			Name:                "Test Type",
			Description:         "Test description",
			TypeName:            "test_type",
			Ranked:              true,
			Color:               null.StringFrom("blue"),
			Icon:                null.StringFrom("users"),
			UseNameAsIdentifier: true,
			Categories:          []string{"Category1", "Category2"},
			Attributes:          []*Attribute{},
		}
	})

	It("creates a basic catalog type model with no attributes", func() {
		base, enumTypes := MarshalType(output)

		Expect(base).NotTo(BeNil())
		Expect(base.Name).To(Equal("Test Type"))
		Expect(base.Description).To(Equal("Test description"))
		Expect(base.TypeName).To(Equal("test_type"))
		Expect(base.Ranked).To(BeTrue())
		Expect(base.Color).To(PointTo(Equal("blue")))
		Expect(base.Icon).To(PointTo(Equal("users")))
		Expect(base.UseNameAsIdentifier).To(BeTrue())
		Expect(base.Categories).To(Equal([]string{"Category1", "Category2"}))
		Expect(base.Attributes).To(HaveLen(0))
		Expect(enumTypes).To(HaveLen(0))
	})

	It("includes standard attributes with API mode", func() {
		output.Attributes = []*Attribute{
			{
				ID:   "attr1",
				Name: "String Attribute",
				Type: null.StringFrom("String"),
			},
			{
				ID:   "attr2",
				Name: "Number Attribute",
				Type: null.StringFrom("Number"),
			},
		}

		base, enumTypes := MarshalType(output)

		Expect(base.Attributes).To(HaveLen(2))
		Expect(base.Attributes[0].Id).To(PointTo(Equal("attr1")))
		Expect(base.Attributes[0].Name).To(Equal("String Attribute"))
		Expect(base.Attributes[0].Type).To(Equal("String"))
		Expect(base.Attributes[0].Mode).To(PointTo(Equal(client.CatalogTypeAttributePayloadV3ModeApi)))

		Expect(base.Attributes[1].Id).To(PointTo(Equal("attr2")))
		Expect(base.Attributes[1].Name).To(Equal("Number Attribute"))
		Expect(base.Attributes[1].Type).To(Equal("Number"))
		Expect(base.Attributes[1].Mode).To(PointTo(Equal(client.CatalogTypeAttributePayloadV3ModeApi)))

		Expect(enumTypes).To(HaveLen(0))
	})

	It("handles array attributes", func() {
		output.Attributes = []*Attribute{
			{
				ID:    "array_attr",
				Name:  "Array Attribute",
				Type:  null.StringFrom("String"),
				Array: true,
			},
		}

		base, _ := MarshalType(output)

		Expect(base.Attributes).To(HaveLen(1))
		Expect(base.Attributes[0].Id).To(PointTo(Equal("array_attr")))
		Expect(base.Attributes[0].Array).To(BeTrue())
	})

	It("processes enum attributes and creates enum types", func() {
		output.Attributes = []*Attribute{
			{
				ID:   "enum_attr",
				Name: "Enum Attribute",
				Enum: &AttributeEnum{
					Name:           "Test Enum",
					Description:    "Test enum description",
					TypeName:       "test_enum",
					EnableBacklink: false,
				},
			},
		}

		base, enumTypes := MarshalType(output)

		Expect(base.Attributes).To(HaveLen(1))
		Expect(base.Attributes[0].Id).To(PointTo(Equal("enum_attr")))
		Expect(base.Attributes[0].Name).To(Equal("Enum Attribute"))
		Expect(base.Attributes[0].Type).To(Equal("test_enum"))

		Expect(enumTypes).To(HaveLen(1))
		Expect(enumTypes[0].Name).To(Equal("Test Enum"))
		Expect(enumTypes[0].Description).To(Equal("Test enum description"))
		Expect(enumTypes[0].TypeName).To(Equal("test_enum"))
		Expect(enumTypes[0].Ranked).To(BeTrue())
		Expect(enumTypes[0].Attributes).To(HaveLen(1)) // Only description attribute, no backlink
		Expect(enumTypes[0].SourceAttribute).ToNot(BeNil())
		Expect(enumTypes[0].SourceAttribute.ID).To(Equal("enum_attr"))
	})

	It("adds backlink attributes to enum types when enabled", func() {
		output.Attributes = []*Attribute{
			{
				ID:   "enum_with_backlink",
				Name: "Enum With Backlink",
				Enum: &AttributeEnum{
					Name:           "Backlinked Enum",
					Description:    "Enum with backlink",
					TypeName:       "backlinked_enum",
					EnableBacklink: true,
				},
			},
		}

		base, enumTypes := MarshalType(output)

		Expect(enumTypes).To(HaveLen(1))
		Expect(enumTypes[0].Attributes).To(HaveLen(2)) // Description + backlink

		backlinkAttr := enumTypes[0].Attributes[1]
		Expect(backlinkAttr.Name).To(Equal("Test Type"))
		Expect(backlinkAttr.Type).To(Equal("test_type"))
		Expect(backlinkAttr.Array).To(BeTrue())
		Expect(backlinkAttr.BacklinkAttribute).To(PointTo(Equal("enum_with_backlink")))
		Expect(backlinkAttr.Mode).To(PointTo(Equal(client.CatalogTypeAttributePayloadV3ModeBacklink)))
		Expect(base.Attributes).To(HaveLen(1))
		Expect(base.Attributes[0].Type).To(Equal(enumTypes[0].TypeName))
		Expect(base.Attributes[0].Mode).To(PointTo(Equal(client.CatalogTypeAttributePayloadV3ModeApi)))
	})

	It("handles backlink attributes", func() {
		output.Attributes = []*Attribute{
			{
				ID:                "backlink_attr",
				Name:              "Backlink Attribute",
				Type:              null.StringFrom("OtherType"),
				BacklinkAttribute: null.StringFrom("source_attr"),
			},
		}

		base, _ := MarshalType(output)

		Expect(base.Attributes).To(HaveLen(1))
		Expect(base.Attributes[0].Id).To(PointTo(Equal("backlink_attr")))
		Expect(base.Attributes[0].Mode).To(PointTo(Equal(client.CatalogTypeAttributePayloadV3ModeBacklink)))
		Expect(base.Attributes[0].BacklinkAttribute).To(PointTo(Equal("source_attr")))
	})

	It("processes path attributes", func() {
		output.Attributes = []*Attribute{
			{
				ID:   "path_attr",
				Name: "Path Attribute",
				Type: null.StringFrom("String"),
				Path: []string{"step1", "step2", "step3"},
			},
		}

		base, _ := MarshalType(output)

		Expect(base.Attributes).To(HaveLen(1))
		Expect(base.Attributes[0].Id).To(PointTo(Equal("path_attr")))
		Expect(base.Attributes[0].Mode).To(PointTo(Equal(client.CatalogTypeAttributePayloadV3ModePath)))
		Expect(base.Attributes[0].Path).ToNot(BeNil())

		pathItems := lo.Map(*base.Attributes[0].Path, func(item client.CatalogTypeAttributePathItemPayloadV3, _ int) string {
			return item.AttributeId
		})
		Expect(pathItems).To(Equal([]string{"step1", "step2", "step3"}))
	})

	It("sets dashboard mode for schema-only attributes", func() {
		output.Attributes = []*Attribute{
			{
				ID:         "dashboard_attr",
				Name:       "Dashboard Attribute",
				Type:       null.StringFrom("String"),
				SchemaOnly: true,
			},
		}

		base, _ := MarshalType(output)

		Expect(base.Attributes).To(HaveLen(1))
		Expect(base.Attributes[0].Id).To(PointTo(Equal("dashboard_attr")))
		Expect(base.Attributes[0].Mode).To(PointTo(Equal(client.CatalogTypeAttributePayloadV3ModeDashboard)))
	})

	It("correctly handles multiple attributes of different types", func() {
		output.Attributes = []*Attribute{
			{
				ID:   "string_attr",
				Name: "String Attribute",
				Type: null.StringFrom("String"),
			},
			{
				ID:   "enum_attr",
				Name: "Enum Attribute",
				Enum: &AttributeEnum{
					Name:           "Test Enum",
					Description:    "Test enum description",
					TypeName:       "test_enum",
					EnableBacklink: true,
				},
			},
			{
				ID:                "backlink_attr",
				Name:              "Backlink Attribute",
				Type:              null.StringFrom("OtherType"),
				BacklinkAttribute: null.StringFrom("source_attr"),
			},
			{
				ID:   "path_attr",
				Name: "Path Attribute",
				Type: null.StringFrom("String"),
				Path: []string{"path1", "path2"},
			},
		}

		base, enumTypes := MarshalType(output)

		Expect(base.Attributes).To(HaveLen(4))
		Expect(enumTypes).To(HaveLen(1))
	})
})
