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
		entries           []source.Entry
		logger            kitlog.Logger
	)

	ctx = context.Background()

	sourceEntry1 := source.Entry{
		"id":               "P123",
		"name":             "Component name",
		"important":        true,
		"importance_score": 100,
		"description":      "A super important component. A structurally integral component tbh.",
		"metadata": map[string]any{
			"namespace": "Infrastructure",
			"aliases":   []string{"oneAlias", "anotherAlias"},
		},
	}
	sourceEntry2 := source.Entry{
		"id":               "P123",
		"name":             "Component name",
		"important":        true,
		"importance_score": 100,
		"description":      "A super important component. A structurally integral component tbh.",
		"metadata": map[string]any{
			"namespace": "Infrastructure",
		},
		"aliases": []string{"andAnotherAlias"},
	}

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

	entries = append(entries, sourceEntry1, sourceEntry2)

	logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))

	When("doing the thing with the aliases", func() {
		It("does the right shit", func() {
			res, err := MarshalEntries(ctx, catalogTypeOutput, entries, logger)
			expectedResult := []string{"oneAlias", "anotherAlias"}
			Expect(err).NotTo(HaveOccurred())
			Expect(res[0].Aliases).To(Equal(expectedResult))
		})

	})
})
