package reconcile

import (
	"context"
	"fmt"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/output"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

// UnmatchedEntryError reports a source entry that we couldn't write to, because it matches
// no existing entry of the synced catalog type, or matches more than one.
type UnmatchedEntryError struct {
	TypeName string
	Match    string
	Reason   string
}

func (e UnmatchedEntryError) Error() string {
	return fmt.Sprintf("%s: source entry with match %q %s", e.TypeName, e.Match, e.Reason)
}

// SyncedEntries writes custom aliases and attribute values onto existing entries of a
// catalog type that incident.io syncs.
//
// It never creates or deletes entries, and never sends a name, external ID, aliases or
// rank, so the integration keeps owning those. Existing entries that no source entry
// matches are left alone.
//
// A source entry that matches no entry, or more than one, can't be written. We skip it,
// write the rest, and return it in unmatched so the caller can fail the run once
// everything else has synced. err is for failures that stop the sync of this type.
func SyncedEntries(ctx context.Context, logger kitlog.Logger, cl EntriesClient, outputType *output.SyncedTypeOutput, catalogType *client.CatalogTypeV3, attributes []*output.Attribute, models []*output.SyncedEntryModel, progress *EntriesProgress, pageSize int) (unmatched []error, err error) {
	logger = kitlog.With(logger,
		"catalog_type_id", catalogType.Id,
		"catalog_type_name", catalogType.TypeName,
	)
	if progress == nil {
		progress = new(EntriesProgress)
	}

	logger.Log("msg", "listing existing entries")
	catalogType, entries, err := cl.GetEntries(ctx, catalogType.Id, pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "listing entries")
	}

	matcher := newEntryMatcher(entries)
	matchedBy := map[string]string{} // entry ID -> match value that claimed it

	toUpdate := []client.PartialEntryPayloadV3{}
	for _, model := range models {
		entry, reason := matcher.find(model.Match)
		if entry == nil {
			unmatched = append(unmatched, UnmatchedEntryError{
				TypeName: catalogType.TypeName, Match: model.Match, Reason: reason,
			})
			continue
		}

		// Two source entries writing to one entry would fight over its custom aliases and
		// attributes, so the first one wins.
		if previous, ok := matchedBy[entry.Id]; ok {
			unmatched = append(unmatched, UnmatchedEntryError{
				TypeName: catalogType.TypeName, Match: model.Match,
				Reason: fmt.Sprintf("matches entry %q (id=%s), which source entry with match %q already matched", entry.Name, entry.Id, previous),
			})
			continue
		}
		matchedBy[entry.Id] = model.Match

		var customAliases *[]string
		customAliasesSame := true
		if model.CustomAliases != nil {
			customAliases = lo.ToPtr(normaliseCustomAliases(model.CustomAliases, syncedAliasesOf(*entry)))
			customAliasesSame = lo.ElementsMatch(*customAliases, entry.CustomAliases)
		}

		if customAliasesSame && attributesAreSame(entry.AttributeValues, model.AttributeValues, attributes) {
			logger.Log("msg", "catalog entry has not changed, not updating", "entry_id", entry.Id)
			continue
		}

		logger.Log("msg", "catalog entry has changed, scheduling for update", "entry_id", entry.Id)
		toUpdate = append(toUpdate, client.PartialEntryPayloadV3{
			EntryId:         entry.Id,
			CustomAliases:   customAliases,
			AttributeValues: model.AttributeValues,
		})
	}

	logger.Log("msg", fmt.Sprintf("found %d entries that need updating", len(toUpdate)))
	if onStart := progress.OnUpdateStart; onStart != nil {
		onStart(len(toUpdate))
	}

	// Always send update_attributes, even when empty. Without it, the API treats the
	// attribute values as the entry's complete set and clears every attribute an API key
	// can write that we didn't send, including ones set in the dashboard.
	updateAttributes := lo.ToPtr(lo.Map(attributes, func(attr *output.Attribute, _ int) string { return attr.ID }))

	// Process batches sequentially to respect rate limits.
	for _, batch := range lo.Chunk(toUpdate, 100) {
		logger.Log("msg", fmt.Sprintf("bulk updating %d catalog entries", len(batch)))

		if err := cl.BulkUpdate(ctx, catalogType.Id, batch, updateAttributes); err != nil {
			return unmatched, explainBulkUpdateError(err, catalogType, len(batch))
		}

		if onProgress := progress.OnUpdateProgress; onProgress != nil {
			for range batch {
				onProgress()
			}
		}
	}

	return unmatched, nil
}

// explainBulkUpdateError adds what to do about the errors we expect a customer to hit.
func explainBulkUpdateError(err error, catalogType *client.CatalogTypeV3, count int) error {
	err = errors.Wrap(err, fmt.Sprintf("unable to bulk update %d catalog entries", count))

	// Team membership is resolved through the entries of some types, such as User, so a
	// custom alias on one of them can change who belongs to a team.
	if strings.Contains(err.Error(), "team.memberships_update") {
		return errors.Wrap(err, fmt.Sprintf(
			"changing custom aliases on %s entries can change who belongs to a team, so the API key needs the "+
				"team.memberships_update scope (the \"Can manage team memberships and edit catalog entries\" role)",
			catalogType.TypeName))
	}

	return err
}

// entryMatcher finds the existing entry a source entry refers to, trying its ID, then its
// external ID, then its name, then its synced aliases.
//
// Custom aliases are left out on purpose: this output sets them, so matching on them
// would let a stale custom alias decide which entry it writes to.
type entryMatcher struct {
	tiers []lo.Tuple2[string, map[string][]*client.CatalogEntryV3]
}

func newEntryMatcher(entries []client.CatalogEntryV3) *entryMatcher {
	byID := map[string][]*client.CatalogEntryV3{}
	byExternalID := map[string][]*client.CatalogEntryV3{}
	byName := map[string][]*client.CatalogEntryV3{}
	byAlias := map[string][]*client.CatalogEntryV3{}

	for idx := range entries {
		entry := &entries[idx]
		byID[entry.Id] = append(byID[entry.Id], entry)
		if entry.ExternalId != nil && *entry.ExternalId != "" {
			byExternalID[*entry.ExternalId] = append(byExternalID[*entry.ExternalId], entry)
		}
		byName[entry.Name] = append(byName[entry.Name], entry)
		for _, alias := range lo.Uniq(syncedAliasesOf(*entry)) {
			byAlias[alias] = append(byAlias[alias], entry)
		}
	}

	return &entryMatcher{
		tiers: []lo.Tuple2[string, map[string][]*client.CatalogEntryV3]{
			lo.T2("ID", byID),
			lo.T2("external ID", byExternalID),
			lo.T2("name", byName),
			lo.T2("alias", byAlias),
		},
	}
}

// find returns the single entry that value refers to, or a reason why there isn't one.
func (m *entryMatcher) find(value string) (*client.CatalogEntryV3, string) {
	if value == "" {
		return nil, "has an empty match value"
	}

	for _, tier := range m.tiers {
		candidates := tier.B[value]
		switch len(candidates) {
		case 0:
			continue
		case 1:
			return candidates[0], ""
		default:
			return nil, fmt.Sprintf("is ambiguous: it is the %s of %d entries (%s)", tier.A, len(candidates),
				strings.Join(lo.Map(candidates, func(entry *client.CatalogEntryV3, _ int) string {
					return fmt.Sprintf("%q id=%s", entry.Name, entry.Id)
				}), ", "))
		}
	}

	return nil, "matches no entry by ID, external ID, name or alias. The importer never creates entries in a synced type, so check the entry exists in incident.io"
}

// syncedAliasesOf returns the aliases the integration syncs onto an entry. The API lists
// custom aliases in aliases too.
func syncedAliasesOf(entry client.CatalogEntryV3) []string {
	return lo.Without(entry.Aliases, entry.CustomAliases...)
}

// normaliseCustomAliases matches how the API stores custom aliases, so an unchanged list
// doesn't look like a change. It also drops any alias the integration already syncs onto
// the entry, which the API rejects: the entry already answers to it, and if the
// integration stops syncing it, the next run adds it back as a custom alias.
func normaliseCustomAliases(customAliases []string, syncedAliases []string) []string {
	return lo.Without(lo.Uniq(lo.Compact(lo.Map(customAliases, func(alias string, _ int) string {
		return strings.TrimSpace(alias)
	}))), syncedAliases...)
}
