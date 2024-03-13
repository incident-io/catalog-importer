package output

import (
	"context"
	"os"

	kitlog "github.com/go-kit/log"
	"github.com/incident-io/catalog-importer/v2/source"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Marshalling data", func() {
	var (
		ctx               context.Context
		catalogTypeOutput *Output
		logger            kitlog.Logger
	)

	ctx = context.Background()

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

	logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))

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
