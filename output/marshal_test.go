package output

import (
	"context"
	"os"

	"github.com/davecgh/go-spew/spew"
	kitlog "github.com/go-kit/log"
	"github.com/incident-io/catalog-importer/v2/source"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/guregu/null.v3"
)

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

				res, err := MarshalEntries(ctx, catalogTypeOutput, entries, logger)

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
				res, err := MarshalEntries(ctx, catalogTypeOutput, entries, logger)
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

		When("Marshalling attributes that aren't present in the source entry", func() {
			It("doesn't include the attribute in the resulting entry", func() {
				sourceEntry := source.Entry{
					"id":          "P1235",
					"name":        "Component name 2",
					"description": "A super important component. A structurally integral component tbh.",
				}
				entries := []source.Entry{sourceEntry}
				res, err := MarshalEntries(ctx, catalogTypeOutput, entries, logger)
				Expect(err).NotTo(HaveOccurred())
				spew.Dump(res)
				Expect(res[0].AttributeValues).To(BeEmpty())
			})
		})
	})
})
