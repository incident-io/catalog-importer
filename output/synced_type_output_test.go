package output_test

import (
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/output"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/guregu/null.v3"
)

var _ = Describe("SyncedTypeOutput", func() {
	var syncedOutput output.SyncedTypeOutput
	BeforeEach(func() {
		syncedOutput = output.SyncedTypeOutput{
			TypeName: "Schedule",
			Source: output.SyncedTypeSourceConfig{
				Match:         "$.schedule",
				CustomAliases: []string{"$.handle"},
			},
			Attributes: []*output.SyncedTypeAttribute{
				{ID: "runbook"},
			},
		}
	})

	Describe("Validate", func() {
		It("accepts a valid output", func() {
			Expect(syncedOutput.Validate()).To(Succeed())
		})

		It("rejects Custom types, which are for outputs", func() {
			syncedOutput.TypeName = `Custom["Service"]`
			Expect(syncedOutput.Validate()).To(MatchError(ContainSubstring("use outputs for Custom types")))
		})

		It("requires a match expression", func() {
			syncedOutput.Source.Match = ""
			Expect(syncedOutput.Validate()).To(MatchError(ContainSubstring("match")))
		})

		It("accepts an empty custom alias list, which clears them", func() {
			syncedOutput.Source.CustomAliases = []string{}
			syncedOutput.Attributes = nil
			Expect(syncedOutput.Validate()).To(Succeed())
		})

		It("rejects an output with nothing to write", func() {
			syncedOutput.Source.CustomAliases = nil
			syncedOutput.Attributes = nil
			Expect(syncedOutput.Validate()).To(MatchError(ContainSubstring("nothing to write")))
		})

		It("rejects an attribute listed twice", func() {
			syncedOutput.Attributes = append(syncedOutput.Attributes, &output.SyncedTypeAttribute{ID: "runbook"})
			Expect(syncedOutput.Validate()).To(MatchError(ContainSubstring("listed more than once")))
		})
	})

	Describe("CompileExpressions", func() {
		It("reports each invalid expression", func() {
			syncedOutput.Source.Match = "$.schedule +"
			syncedOutput.Attributes[0].Source = null.StringFrom("$.runbook +")

			errs := syncedOutput.CompileExpressions()
			Expect(errs).To(HaveLen(2))
			Expect(errs[0]).To(MatchError(ContainSubstring("source.match")))
			Expect(errs[1]).To(MatchError(ContainSubstring("attributes.runbook")))
		})
	})

	Describe("ResolveAttributes", func() {
		var catalogType *client.CatalogTypeV3
		BeforeEach(func() {
			catalogType = &client.CatalogTypeV3{
				TypeName: "Schedule",
				Schema: client.CatalogTypeSchemaV3{
					Attributes: []client.CatalogTypeAttributeV3{
						{Id: "users", Name: "Users", Type: "User", Array: true, Mode: client.CatalogTypeAttributeV3ModeInternal},
						{Id: "runbook", Name: "Runbook", Type: "String", Mode: client.CatalogTypeAttributeV3ModeDashboard},
						{Id: "tier", Name: "Tier", Type: "Number", Mode: client.CatalogTypeAttributeV3ModeApi},
					},
				},
			}
		})

		It("takes the type and array setting from the schema", func() {
			syncedOutput.Attributes = []*output.SyncedTypeAttribute{
				{ID: "runbook"},
				{ID: "tier", Source: null.StringFrom("$.service_tier")},
			}

			attributes, err := syncedOutput.ResolveAttributes(catalogType)
			Expect(err).NotTo(HaveOccurred())
			Expect(attributes).To(HaveLen(2))
			Expect(attributes[0].ID).To(Equal("runbook"))
			Expect(attributes[0].Type).To(Equal(null.StringFrom("String")))
			Expect(attributes[0].SourceExpression()).To(Equal("$.runbook"))
			Expect(attributes[1].Type).To(Equal(null.StringFrom("Number")))
			Expect(attributes[1].SourceExpression()).To(Equal("$.service_tier"))
		})

		It("rejects an attribute that isn't on the type", func() {
			syncedOutput.Attributes = []*output.SyncedTypeAttribute{{ID: "missing"}}

			_, err := syncedOutput.ResolveAttributes(catalogType)
			Expect(err).To(MatchError(And(
				ContainSubstring(`attribute "missing" does not exist on Schedule`),
				ContainSubstring("runbook (Runbook), tier (Tier)"),
			)))
		})

		It("rejects an attribute that incident.io syncs", func() {
			syncedOutput.Attributes = []*output.SyncedTypeAttribute{{ID: "users"}}

			_, err := syncedOutput.ResolveAttributes(catalogType)
			Expect(err).To(MatchError(ContainSubstring("is set by incident.io")))
		})
	})
})
