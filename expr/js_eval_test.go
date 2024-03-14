package expr

import (
	"context"
	"os"

	kitlog "github.com/go-kit/log"
	"github.com/incident-io/catalog-importer/v2/source"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Javascript evaluation", func() {
	var (
		ctx                  context.Context
		sourceEntry          source.Entry
		sourceEntryWithArray source.Entry
		logger               kitlog.Logger
	)

	ctx = context.Background()

	sourceEntry = source.Entry{
		"id":               "P123",
		"name":             "Component name",
		"important":        true,
		"importance_score": 100,
		"description":      "A super important component. A structurally integral component tbh.",
		"metadata": map[string]any{
			"namespace": "Infrastructure",
		},
	}

	sourceEntryWithArray = source.Entry{
		"id":          "P124",
		"name":        "Component name with an array",
		"description": "This one has multiple values, which is kinda neat",
		"array":       true,
		"domains":     []string{"something", "something else"},
	}

	logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))

	When("parsing attribute sources", func() {
		It("returns the correct top-level attribute", func() {
			topLevelSrc := "$.name"
			evaluatedResult, err := EvaluateSingleValue[string](ctx, topLevelSrc, sourceEntry, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(*evaluatedResult).To(Equal(sourceEntry["name"]))
		})

		It("returns a bool as expected", func() {
			topLevelSrc := "$.important"
			evaluatedResult, err := EvaluateSingleValue[bool](ctx, topLevelSrc, sourceEntry, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(*evaluatedResult).To(Equal(sourceEntry["important"]))
		})

		It("returns a number as expected", func() {
			topLevelSrc := "$.importance_score"
			evaluatedResult, err := EvaluateSingleValue[int](ctx, topLevelSrc, sourceEntry, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(*evaluatedResult).To(Equal(sourceEntry["importance_score"]))
		})

		It("returns a string as expected", func() {
			topLevelSrc := "$.description"
			evaluatedResult, err := EvaluateSingleValue[string](ctx, topLevelSrc, sourceEntry, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(*evaluatedResult).To(Equal(sourceEntry["description"]))
		})

		It("does not parse a value if given the wrong type", func() {
			topLevelSrc := "$.description"
			_, err := EvaluateSingleValue[int](ctx, topLevelSrc, sourceEntry, logger)
			Expect(err).To(HaveOccurred(), "could not convert result of string to int")
		})

		It("returns nil if the type is not supported", func() {
			topLevelSrc := "$.metadata"
			evaluatedResult, err := EvaluateSingleValue[string](ctx, topLevelSrc, sourceEntry, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(*evaluatedResult).To(Equal(""))
		})
	})

	It("manipulates string values as expected", func() {
		topLevelSrc := "$.name.replace('Component', 'Replacement')"
		evaluatedResult, err := EvaluateSingleValue[string](ctx, topLevelSrc, sourceEntry, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(*evaluatedResult).To(Equal("Replacement name"))
	})

	It("parses nested values as expected", func() {
		topLevelSrc := "$.metadata.namespace"
		evaluatedResult, err := EvaluateSingleValue[string](ctx, topLevelSrc, sourceEntry, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(*evaluatedResult).To(Equal(sourceEntry["metadata"].(map[string]any)["namespace"]))
	})

	It("handles possible null values with _.get", func() {
		nestedSrc := "_.get($.metadata, \"badKey\", \"default value\")"
		evaluatedResult, err := EvaluateSingleValue[string](ctx, nestedSrc, sourceEntry, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(*evaluatedResult).To(Equal("default value"))
	})

	When("parsing array values", func() {
		It("returns an error if the input is not an array", func() {
			topLevelSrc := "$.name"
			entryName, ok := sourceEntryWithArray["name"].(string)
			Expect(ok).To(BeTrue())
			expectedResult := []string{entryName}
			evaluatedResult, err := EvaluateArray[string](ctx, topLevelSrc, sourceEntryWithArray, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(evaluatedResult).To(Equal(expectedResult))
		})

		It("works as expected when given actual array input", func() {
			topLevelSrc := "$.domains"
			evaluatedResult, err := EvaluateArray[string](ctx, topLevelSrc, sourceEntryWithArray, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(evaluatedResult).To(Equal(sourceEntryWithArray["domains"]))
		})
	})

	When("sending invalid source javascript", func() {
		It("returns nothing if I send a key that isn't present on the entry", func() {
			topLevelSrc := "$.ghostkey"
			evaluatedResult, err := EvaluateSingleValue[string](ctx, topLevelSrc, sourceEntry, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(evaluatedResult).To(BeNil())
		})

		It("returns nil if my JS is invalid", func() {
			topLevelSrc := "$badKey"
			evaluatedResult, err := EvaluateArray[string](ctx, topLevelSrc, sourceEntryWithArray, logger)
			Expect(err).NotTo(HaveOccurred())
			// Expecting an array with an empty string here, as that is the empty state for this function
			Expect(evaluatedResult).To(BeNil())
		})
	})

})
