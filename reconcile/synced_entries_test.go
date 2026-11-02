package reconcile_test

import (
	"context"
	"errors"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/output"
	"github.com/incident-io/catalog-importer/v2/reconcile"
	"github.com/samber/lo"
	"gopkg.in/guregu/null.v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SyncedEntries", func() {
	var (
		ctx    context.Context
		logger kitlog.Logger

		existingEntries  []client.CatalogEntryV3
		bulkUpdates      [][]client.PartialEntryPayloadV3
		updateAttributes []*[]string
		bulkUpdateErr    error
		mockClient       reconcile.EntriesClient

		catalogType  *client.CatalogTypeV3
		syncedOutput *output.SyncedTypeOutput
		attributes   []*output.Attribute
		models       []*output.SyncedEntryModel
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = kitlog.NewNopLogger()

		bulkUpdates = nil
		updateAttributes = nil
		bulkUpdateErr = nil

		existingEntries = []client.CatalogEntryV3{
			{
				Id:            "01SCHEDULEPRIMARY",
				Name:          "Primary on-call",
				ExternalId:    lo.ToPtr("ext-primary"),
				Aliases:       []string{"primary-synced", "old-custom"},
				CustomAliases: []string{"old-custom"},
				AttributeValues: map[string]client.CatalogEntryEngineParamBindingV3{
					"runbook": {Value: &client.CatalogEntryEngineParamBindingValueV3{Literal: lo.ToPtr("https://example.com/old")}},
				},
			},
			{
				Id:              "01SCHEDULESECONDARY",
				Name:            "Secondary on-call",
				Aliases:         []string{"secondary-synced"},
				CustomAliases:   []string{},
				AttributeValues: map[string]client.CatalogEntryEngineParamBindingV3{},
			},
		}

		mockClient = reconcile.EntriesClient{
			GetEntries: func(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV3, []client.CatalogEntryV3, error) {
				return catalogType, existingEntries, nil
			},
			Create: func(ctx context.Context, payload client.CatalogCreateEntryPayloadV3) (*client.CatalogEntryV3, error) {
				Fail("synced entries must never be created")
				return nil, nil
			},
			Delete: func(ctx context.Context, entry *client.CatalogEntryV3) error {
				Fail("synced entries must never be deleted")
				return nil
			},
			BulkUpdate: func(ctx context.Context, catalogTypeID string, entries []client.PartialEntryPayloadV3, attrs *[]string) error {
				if bulkUpdateErr != nil {
					return bulkUpdateErr
				}
				bulkUpdates = append(bulkUpdates, entries)
				updateAttributes = append(updateAttributes, attrs)
				return nil
			},
		}

		catalogType = &client.CatalogTypeV3{Id: "01SCHEDULETYPE", TypeName: "Schedule"}
		syncedOutput = &output.SyncedTypeOutput{
			TypeName: "Schedule",
			Source:   output.SyncedTypeSourceConfig{Match: "$.schedule", CustomAliases: []string{"$.handle"}},
		}
		attributes = []*output.Attribute{
			{ID: "runbook", Name: "Runbook", Type: null.StringFrom("String")},
		}
		models = nil
	})

	run := func() ([]error, error) {
		return reconcile.SyncedEntries(ctx, logger, mockClient, syncedOutput, catalogType, attributes, models, nil, 100)
	}
	runb := func(value string) map[string]client.CatalogEngineParamBindingPayloadV3 {
		return map[string]client.CatalogEngineParamBindingPayloadV3{
			"runbook": {Value: &client.CatalogEngineParamBindingValuePayloadV3{Literal: lo.ToPtr(value)}},
		}
	}
	updatedEntries := func() []client.PartialEntryPayloadV3 {
		return lo.Flatten(bulkUpdates)
	}

	It("writes only custom aliases and declared attributes to the matched entry", func() {
		models = []*output.SyncedEntryModel{
			{Match: "Primary on-call", CustomAliases: []string{"primary", "ops"}, AttributeValues: runb("https://example.com/new")},
		}

		unmatched, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(unmatched).To(BeEmpty())

		Expect(updatedEntries()).To(Equal([]client.PartialEntryPayloadV3{
			{
				EntryId:         "01SCHEDULEPRIMARY",
				CustomAliases:   lo.ToPtr([]string{"primary", "ops"}),
				AttributeValues: runb("https://example.com/new"),
			},
		}))
		Expect(updateAttributes).To(Equal([]*[]string{lo.ToPtr([]string{"runbook"})}))
	})

	DescribeTable("matches an entry",
		func(match string) {
			models = []*output.SyncedEntryModel{
				{Match: match, CustomAliases: []string{"primary"}, AttributeValues: runb("https://example.com/old")},
			}

			unmatched, err := run()
			Expect(err).NotTo(HaveOccurred())
			Expect(unmatched).To(BeEmpty())
			Expect(updatedEntries()).To(HaveLen(1))
			Expect(updatedEntries()[0].EntryId).To(Equal("01SCHEDULEPRIMARY"))
		},
		Entry("by ID", "01SCHEDULEPRIMARY"),
		Entry("by external ID", "ext-primary"),
		Entry("by name", "Primary on-call"),
		Entry("by synced alias", "primary-synced"),
	)

	It("doesn't match on custom aliases, which this output sets", func() {
		models = []*output.SyncedEntryModel{
			{Match: "old-custom", CustomAliases: []string{}, AttributeValues: runb("x")},
		}

		unmatched, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(unmatched).To(HaveLen(1))
		Expect(updatedEntries()).To(BeEmpty())
	})

	It("leaves entries alone that no source entry matches", func() {
		models = []*output.SyncedEntryModel{
			{Match: "Secondary on-call", CustomAliases: []string{"secondary"}, AttributeValues: runb("x")},
		}

		_, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedEntries()).To(HaveLen(1))
		Expect(updatedEntries()[0].EntryId).To(Equal("01SCHEDULESECONDARY"))
	})

	It("skips entries that haven't changed", func() {
		models = []*output.SyncedEntryModel{
			{Match: "Primary on-call", CustomAliases: []string{" old-custom ", "old-custom"}, AttributeValues: runb("https://example.com/old")},
		}

		_, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(bulkUpdates).To(BeEmpty())
	})

	It("clears custom aliases when the source has none", func() {
		models = []*output.SyncedEntryModel{
			{Match: "Primary on-call", CustomAliases: []string{}, AttributeValues: runb("https://example.com/old")},
		}

		_, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedEntries()).To(HaveLen(1))
		Expect(updatedEntries()[0].CustomAliases).To(Equal(lo.ToPtr([]string{})))
	})

	It("drops custom aliases the integration already syncs, which the API rejects", func() {
		models = []*output.SyncedEntryModel{
			{Match: "Primary on-call", CustomAliases: []string{"primary-synced", "primary"}, AttributeValues: runb("https://example.com/old")},
		}

		_, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedEntries()[0].CustomAliases).To(Equal(lo.ToPtr([]string{"primary"})))
	})

	It("doesn't send custom aliases when the output doesn't manage them", func() {
		models = []*output.SyncedEntryModel{
			{Match: "Primary on-call", AttributeValues: runb("https://example.com/new")},
		}

		_, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedEntries()).To(HaveLen(1))
		Expect(updatedEntries()[0].CustomAliases).To(BeNil())
	})

	It("sends an empty update_attributes when no attributes are declared, so no attribute is cleared", func() {
		attributes = nil
		models = []*output.SyncedEntryModel{
			{Match: "Primary on-call", CustomAliases: []string{"primary"}, AttributeValues: map[string]client.CatalogEngineParamBindingPayloadV3{}},
		}

		_, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(updateAttributes).To(Equal([]*[]string{lo.ToPtr([]string{})}))
	})

	It("reports unmatched and ambiguous source entries, and still writes the rest", func() {
		existingEntries = append(existingEntries, client.CatalogEntryV3{
			Id: "01SCHEDULEDUPLICATE", Name: "Secondary on-call", Aliases: []string{}, CustomAliases: []string{},
		})
		models = []*output.SyncedEntryModel{
			{Match: "Deleted schedule", CustomAliases: []string{"gone"}},
			{Match: "Secondary on-call", CustomAliases: []string{"secondary"}},
			{Match: "", CustomAliases: []string{"blank"}},
			{Match: "Primary on-call", CustomAliases: []string{"primary"}, AttributeValues: runb("https://example.com/old")},
		}

		unmatched, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(unmatched).To(HaveLen(3))
		Expect(unmatched[0]).To(MatchError(ContainSubstring(`Schedule: source entry with match "Deleted schedule" matches no entry`)))
		Expect(unmatched[1]).To(MatchError(ContainSubstring("is ambiguous: it is the name of 2 entries")))
		Expect(unmatched[2]).To(MatchError(ContainSubstring("empty match value")))

		Expect(updatedEntries()).To(HaveLen(1))
		Expect(updatedEntries()[0].EntryId).To(Equal("01SCHEDULEPRIMARY"))
	})

	It("lets only the first source entry write to an entry two of them match", func() {
		models = []*output.SyncedEntryModel{
			{Match: "Primary on-call", CustomAliases: []string{"first"}},
			{Match: "ext-primary", CustomAliases: []string{"second"}},
		}

		unmatched, err := run()
		Expect(err).NotTo(HaveOccurred())
		Expect(unmatched).To(HaveLen(1))
		Expect(unmatched[0]).To(MatchError(ContainSubstring(`already matched`)))
		Expect(updatedEntries()).To(HaveLen(1))
		Expect(updatedEntries()[0].CustomAliases).To(Equal(lo.ToPtr([]string{"first"})))
	})

	It("explains the missing team memberships scope", func() {
		bulkUpdateErr = errors.New(`status 403: {"errors":[{"message":"missing required scope team.memberships_update"}]}`)
		models = []*output.SyncedEntryModel{
			{Match: "Primary on-call", CustomAliases: []string{"primary"}},
		}

		_, err := run()
		Expect(err).To(MatchError(ContainSubstring("the API key needs the team.memberships_update scope")))
	})
})
