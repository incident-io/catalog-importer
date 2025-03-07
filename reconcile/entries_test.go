package reconcile_test

import (
	"context"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/output"
	"github.com/incident-io/catalog-importer/v2/reconcile"
	"github.com/samber/lo"
	"gopkg.in/guregu/null.v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Entries", func() {
	var (
		ctx    context.Context
		logger kitlog.Logger
	)
	BeforeEach(func() {
		ctx = context.Background()
		logger = kitlog.NewNopLogger()
	})

	// Set up a mock client
	type updatedEntry struct {
		id      string
		payload client.CatalogUpdateEntryPayloadV3
	}
	var (
		mockClient      reconcile.EntriesClient
		existingEntries []client.CatalogEntryV3
		createdEntries  []client.CatalogCreateEntryPayloadV3
		updatedEntries  []updatedEntry
		deletedEntries  []string
	)
	BeforeEach(func() {
		// Reset
		existingEntries = []client.CatalogEntryV3{}
		createdEntries = []client.CatalogCreateEntryPayloadV3{}
		updatedEntries = []updatedEntry{}
		deletedEntries = []string{}

		attrValuesFromPayload := func(payload map[string]client.CatalogEngineParamBindingPayloadV3) map[string]client.CatalogEntryEngineParamBindingV3 {
			attributeValues := map[string]client.CatalogEntryEngineParamBindingV3{}
			for k, v := range payload {
				res := client.CatalogEntryEngineParamBindingV3{}
				if v.Value != nil {
					res.Value = &client.CatalogEntryEngineParamBindingValueV3{
						Label:   lo.FromPtrOr(v.Value.Literal, ""),
						Literal: v.Value.Literal,
					}
				}
				if v.ArrayValue != nil {
					res.ArrayValue = &[]client.CatalogEntryEngineParamBindingValueV3{}
					for _, arrayVal := range *v.ArrayValue {
						*res.ArrayValue = append(*res.ArrayValue, client.CatalogEntryEngineParamBindingValueV3{
							Label:   lo.FromPtrOr(arrayVal.Literal, ""),
							Literal: arrayVal.Literal,
						})
					}
				}
				attributeValues[k] = res
			}

			return attributeValues
		}

		// Setup a mock client
		mockClient = reconcile.EntriesClient{
			GetEntries: func(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV3, []client.CatalogEntryV3, error) {
				return &client.CatalogTypeV3{
					Id:       "type-123",
					TypeName: "Test Type",
				}, existingEntries, nil
			},
			Create: func(ctx context.Context, payload client.CatalogCreateEntryPayloadV3) (*client.CatalogEntryV3, error) {
				createdEntries = append(createdEntries, payload)

				return &client.CatalogEntryV3{
					Id:              "entry-" + *payload.ExternalId,
					Name:            payload.Name,
					ExternalId:      payload.ExternalId,
					Aliases:         *payload.Aliases,
					Rank:            *payload.Rank,
					AttributeValues: attrValuesFromPayload(payload.AttributeValues),
				}, nil
			},
			Delete: func(ctx context.Context, entry *client.CatalogEntryV3) error {
				deletedEntries = append(deletedEntries, entry.Id)
				return nil
			},
			Update: func(ctx context.Context, entry *client.CatalogEntryV3, payload client.CatalogUpdateEntryPayloadV3) (*client.CatalogEntryV3, error) {
				updatedEntries = append(updatedEntries, updatedEntry{
					id:      entry.Id,
					payload: payload,
				})
				return &client.CatalogEntryV3{
					Id:              entry.Id,
					Name:            payload.Name,
					ExternalId:      payload.ExternalId,
					Aliases:         lo.FromPtrOr(payload.Aliases, []string{}),
					Rank:            *payload.Rank,
					AttributeValues: attrValuesFromPayload(payload.AttributeValues),
				}, nil
			},
		}
	})

	// Inputs to the reconcile function
	var (
		catalogType *client.CatalogTypeV3
		outputType  *output.Output
		entryModels []*output.CatalogEntryModel
	)
	reconcile := func() error {
		return reconcile.Entries(ctx, logger, mockClient, outputType, catalogType, entryModels, nil, 100)
	}
	mustReconcile := func() {
		err := reconcile()
		Expect(err).NotTo(HaveOccurred())
	}

	When("no entries exist", func() {
		BeforeEach(func() {
			// Setup test data
			catalogType = &client.CatalogTypeV3{
				Id:       "type-123",
				TypeName: "Test Type",
			}

			outputType = &output.Output{
				Attributes: []*output.Attribute{
					{
						ID:   "attr1",
						Name: "Attribute 1",
					},
				},
			}

			entryModels = []*output.CatalogEntryModel{
				{
					Name:       "Entry 1",
					ExternalID: "ext-1",
					Rank:       100,
					Aliases:    []string{"alias1", "alias2"},
					AttributeValues: map[string]client.CatalogEngineParamBindingPayloadV3{
						"attr1": {
							Value: &client.CatalogEngineParamBindingValuePayloadV3{
								Literal: lo.ToPtr("option1"),
							},
						},
					},
				},
				{
					Name:       "Entry 2",
					ExternalID: "ext-2",
					Rank:       200,
					Aliases:    []string{"alias3"},
					AttributeValues: map[string]client.CatalogEngineParamBindingPayloadV3{
						"attr1": {
							Value: &client.CatalogEngineParamBindingValuePayloadV3{
								Literal: lo.ToPtr("option2"),
							},
						},
					},
				},
			}

		})

		It("creates all entries", func() {
			mustReconcile()

			// Verify that all entries were created
			Expect(createdEntries).To(HaveLen(2))

			// Verify the entries were created with the right data
			Expect(createdEntries[0].Name).To(Equal("Entry 1"))
			Expect(*createdEntries[0].ExternalId).To(Equal("ext-1"))
			Expect(*createdEntries[0].Rank).To(Equal(int32(100)))
			Expect(*createdEntries[0].Aliases).To(Equal([]string{"alias1", "alias2"}))

			Expect(createdEntries[1].Name).To(Equal("Entry 2"))
			Expect(*createdEntries[1].ExternalId).To(Equal("ext-2"))
			Expect(*createdEntries[1].Rank).To(Equal(int32(200)))
			Expect(*createdEntries[1].Aliases).To(Equal([]string{"alias3"}))
		})
	})

	When("entries need to be deleted", func() {
		BeforeEach(func() {
			// Setup test data
			catalogType = &client.CatalogTypeV3{
				Id:       "type-123",
				TypeName: "Test Type",
			}

			outputType = &output.Output{
				Attributes: []*output.Attribute{
					{
						ID:   "attr1",
						Name: "Attribute 1",
					},
				},
			}

			existingEntries = []client.CatalogEntryV3{
				{
					Id:         "entry-1",
					ExternalId: lo.ToPtr("ext-1"),
					Name:       "Entry 1",
					Rank:       100,
				},
				{
					Id:         "entry-2",
					ExternalId: lo.ToPtr("ext-to-delete"),
					Name:       "Entry 2",
					Rank:       200,
				},
				{
					Id:         "entry-3",
					ExternalId: lo.ToPtr("ext-to-delete-2"),
					Name:       "Entry 3",
					Rank:       300,
				},
			}

			entryModels = []*output.CatalogEntryModel{
				{
					Name:       "Entry 1",
					ExternalID: "ext-1",
					Rank:       100,
				},
				// two have been removed!
			}
		})

		It("deletes entries that are no longer in source", func() {
			mustReconcile()

			// Verify that the expected entries were deleted
			Expect(deletedEntries).To(HaveLen(2))

			// Should delete the entry with external ID that's not in models and the one without external ID
			Expect(deletedEntries).To(ConsistOf("entry-2", "entry-3"))
		})
	})

	When("entries need to be updated", func() {
		BeforeEach(func() {
			// Setup test data
			catalogType = &client.CatalogTypeV3{
				Id:       "type-123",
				TypeName: "Test Type",
			}

			outputType = &output.Output{
				Attributes: []*output.Attribute{
					{
						ID:         "attr1",
						Name:       "Attribute 1",
						SchemaOnly: false,
					},
					{
						ID:         "attr2",
						Name:       "Attribute 2 (schema only)",
						SchemaOnly: true,
					},
				},
			}

			existingEntries = []client.CatalogEntryV3{
				{
					Id:         "entry-1",
					ExternalId: lo.ToPtr("ext-1"),
					Name:       "Entry 1",
					Rank:       100,
					Aliases:    []string{"old-alias"},
					AttributeValues: map[string]client.CatalogEntryEngineParamBindingV3{
						"attr1": {
							Value: &client.CatalogEntryEngineParamBindingValueV3{
								Label:   "option-old",
								Literal: lo.ToPtr("option-old"),
							},
						},
					},
				},
				{
					Id:         "entry-2",
					ExternalId: lo.ToPtr("ext-2"),
					Name:       "Unchanged Entry",
					Rank:       200,
					Aliases:    []string{"alias"},
					AttributeValues: map[string]client.CatalogEntryEngineParamBindingV3{
						"attr1": {
							Value: &client.CatalogEntryEngineParamBindingValueV3{
								Label:   "option2",
								Literal: lo.ToPtr("option2"),
							},
						},
					},
				},
			}

			entryModels = []*output.CatalogEntryModel{
				{
					Name:       "New Name", // Changed
					ExternalID: "ext-1",
					Rank:       100,                   // Same
					Aliases:    []string{"new-alias"}, // Changed
					AttributeValues: map[string]client.CatalogEngineParamBindingPayloadV3{
						"attr1": {
							Value: &client.CatalogEngineParamBindingValuePayloadV3{
								Literal: lo.ToPtr("option-new"), // Changed
							},
						},
					},
				},
				{
					Name:       "Unchanged Entry", // Same
					ExternalID: "ext-2",
					Rank:       200,               // Same
					Aliases:    []string{"alias"}, // Same
					AttributeValues: map[string]client.CatalogEngineParamBindingPayloadV3{
						"attr1": {
							Value: &client.CatalogEngineParamBindingValuePayloadV3{
								Literal: lo.ToPtr("option2"), // Same
							},
						},
					},
				},
			}

		})

		It("updates only changed entries", func() {
			mustReconcile()

			// Verify that only changed entries were updated
			Expect(updatedEntries).To(HaveLen(1))

			// Verify the updated entry data
			Expect(updatedEntries[0].id).To(Equal("entry-1"))
			payload := updatedEntries[0].payload
			Expect(payload.Name).To(Equal("New Name"))
			Expect(*payload.ExternalId).To(Equal("ext-1"))
			Expect(*payload.Aliases).To(Equal([]string{"new-alias"}))
			Expect(payload.UpdateAttributes).NotTo(BeNil())
			Expect(*payload.UpdateAttributes).To(ConsistOf("attr1")) // schema-only attribute not included
		})
	})

	When("updating entries with different attribute types", func() {
		BeforeEach(func() {
			// Setup test data
			catalogType = &client.CatalogTypeV3{
				Id:       "type-123",
				TypeName: "Test Type",
			}

			outputType = &output.Output{
				Attributes: []*output.Attribute{
					{
						ID:         "normal_attr",
						Name:       "Normal Attribute",
						SchemaOnly: false,
					},
					{
						ID:         "schema_only_attr",
						Name:       "Schema Only Attribute",
						SchemaOnly: true,
					},
					{
						ID:                "backlink_attr",
						Name:              "Backlink Attribute",
						BacklinkAttribute: null.StringFrom("source_attr"),
					},
					{
						ID:   "path_attr",
						Name: "Path Attribute",
						Path: []string{"step1", "step2"},
					},
				},
			}

			existingEntries = []client.CatalogEntryV3{
				{
					Id:         "entry-1",
					ExternalId: lo.ToPtr("ext-1"),
					Name:       "Test Entry",
					AttributeValues: map[string]client.CatalogEntryEngineParamBindingV3{
						"normal_attr": {
							Value: &client.CatalogEntryEngineParamBindingValueV3{
								Literal: lo.ToPtr("old-value"),
							},
						},
						"schema_only_attr": {
							Value: &client.CatalogEntryEngineParamBindingValueV3{
								Literal: lo.ToPtr("dashboard-value"),
							},
						},
					},
				},
			}

			entryModels = []*output.CatalogEntryModel{
				{
					Name:       "Test Entry Updated",
					ExternalID: "ext-1",
					AttributeValues: map[string]client.CatalogEngineParamBindingPayloadV3{
						"normal_attr": {
							Value: &client.CatalogEngineParamBindingValuePayloadV3{
								Literal: lo.ToPtr("new-value"),
							},
						},
						"schema_only_attr": {
							Value: &client.CatalogEngineParamBindingValuePayloadV3{
								Literal: lo.ToPtr("should-be-ignored"),
							},
						},
						"backlink_attr": {
							Value: &client.CatalogEngineParamBindingValuePayloadV3{
								Literal: lo.ToPtr("should-be-ignored"),
							},
						},
						"path_attr": {
							Value: &client.CatalogEngineParamBindingValuePayloadV3{
								Literal: lo.ToPtr("should-be-ignored"),
							},
						},
					},
				},
			}
		})

		It("only includes payload-eligible attributes in UpdateAttributes", func() {
			mustReconcile()
			
			// Verify that entry was updated
			Expect(updatedEntries).To(HaveLen(1))
			
			// Check that only normal_attr was included in UpdateAttributes
			payload := updatedEntries[0].payload
			Expect(payload.UpdateAttributes).NotTo(BeNil())
			Expect(*payload.UpdateAttributes).To(ConsistOf("normal_attr"))
			
			// Verify that non-included attributes are not in the UpdateAttributes list
			Expect(*payload.UpdateAttributes).NotTo(ContainElement("schema_only_attr"))
			Expect(*payload.UpdateAttributes).NotTo(ContainElement("backlink_attr"))
			Expect(*payload.UpdateAttributes).NotTo(ContainElement("path_attr"))
		})
	})

	When("entries have duplicate external IDs", func() {
		BeforeEach(func() {
			// Setup test data
			catalogType = &client.CatalogTypeV3{
				Id:       "type-123",
				TypeName: "Test Type",
			}

			outputType = &output.Output{
				Attributes: []*output.Attribute{
					{
						ID:   "attr1",
						Name: "Attribute 1",
					},
				},
			}

			entryModels = []*output.CatalogEntryModel{
				{
					Name:       "Entry 1",
					ExternalID: "duplicate-id",
					Rank:       100,
				},
				{
					Name:       "Entry 2 (duplicate)",
					ExternalID: "duplicate-id", // Same as first entry
					Rank:       200,
				},
				{
					Name:       "Entry 3",
					ExternalID: "unique-id",
					Rank:       300,
				},
			}
		})

		It("logs but handles duplicate external IDs", func() {
			mustReconcile()

			// We don't do anything clever here, but we do log a warning
			Expect(createdEntries).To(HaveLen(3))

			// Check that the external IDs are correct
			externalIDs := []string{}
			for _, entry := range createdEntries {
				externalIDs = append(externalIDs, *entry.ExternalId)
			}

			Expect(externalIDs).To(ConsistOf("duplicate-id", "duplicate-id", "unique-id"))
		})
	})
})
