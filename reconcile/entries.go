package reconcile

import (
	"context"
	"fmt"
	"reflect"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/output"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/sourcegraph/conc/pool"
)

type EntriesClient struct {
	GetEntries func(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV3, []client.CatalogEntryV3, error)
	Delete     func(ctx context.Context, entry *client.CatalogEntryV3) error
	Create     func(ctx context.Context, payload client.CatalogCreateEntryPayloadV3) (*client.CatalogEntryV3, error)
	BulkUpdate func(ctx context.Context, catalogTypeID string, entries []client.PartialEntryPayloadV3, updateAttributes *[]string) error
}

// EntriesClientFromClient wraps a real client with hooks that can create, update and delete
// entries. This can be overriden for custom behaviour, such as a dry-run that shouldn't
// actually perform updates.
func EntriesClientFromClient(cl *client.ClientWithResponses) EntriesClient {
	return EntriesClient{
		GetEntries: func(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV3, []client.CatalogEntryV3, error) {
			return GetEntries(ctx, cl, catalogTypeID, pageSize)
		},
		Delete: func(ctx context.Context, entry *client.CatalogEntryV3) error {
			_, err := cl.CatalogV3DestroyEntryWithResponse(ctx, entry.Id)
			if err != nil {
				return err
			}

			return nil
		},
		Create: func(ctx context.Context, payload client.CatalogCreateEntryPayloadV3) (*client.CatalogEntryV3, error) {
			result, err := cl.CatalogV3CreateEntryWithResponse(ctx, payload)
			if err != nil {
				return nil, err
			}

			if result.JSON201 == nil {
				return nil, errors.Errorf(
					`unexpected nil 201 response. Status Code: %d, Content-Type: %s, Bytes Length: %d`,
					result.HTTPResponse.StatusCode,
					result.HTTPResponse.Header.Get("Content-Type"),
					len(result.Body),
				)
			}

			return &result.JSON201.CatalogEntry, nil
		},
		BulkUpdate: func(ctx context.Context, catalogTypeID string, entries []client.PartialEntryPayloadV3, updateAttributes *[]string) error {
			_, err := cl.CatalogV3BulkUpdateEntriesWithResponse(ctx, client.CatalogBulkUpdateEntriesPayloadV3{
				CatalogTypeId:    catalogTypeID,
				Entries:          entries,
				UpdateAttributes: updateAttributes,
			})
			return err
		},
	}
}

type EntriesProgress struct {
	OnDeleteStart    func(total int)
	OnDeleteProgress func()
	OnCreateStart    func(total int)
	OnCreateProgress func()
	OnUpdateStart    func(total int)
	OnUpdateProgress func()
}

func Entries(ctx context.Context, logger kitlog.Logger, cl EntriesClient, outputType *output.Output, catalogType *client.CatalogTypeV3, entryModels []*output.CatalogEntryModel, progress *EntriesProgress, pageSize int) error {
	logger = kitlog.With(logger,
		"catalog_type_id", catalogType.Id,
		"catalog_type_name", catalogType.TypeName,
	)

	// Initialise this as it's easy to deal with if you don't nil check the full struct.
	if progress == nil {
		progress = new(EntriesProgress)
	}

	logger.Log("msg", "listing existing entries")
	catalogType, entries, err := cl.GetEntries(ctx, catalogType.Id, pageSize)
	if err != nil {
		return errors.Wrap(err, "listing entries")
	}

	// Prepare a quick lookup of model by external ID, to power deletion. We only need
	// to store a boolean here, because we only care about the presence of a model by
	// external ID.
	modelsByExternalID := map[string]bool{}
	for _, model := range entryModels {
		// If we encounter two (or more) models with the same external ID, then we should
		// log a warning. The external ID must be unique, so in the next phase we'll only
		// create one catalog entry per external ID.
		_, ok := modelsByExternalID[model.ExternalID]
		if ok {
			logger.Log(
				"msg", "two entries with the same external ID provided, the first will be ignored",
				"external_id", model.ExternalID,
			)
		}

		modelsByExternalID[model.ExternalID] = true
	}

	{
		toDelete := []client.CatalogEntryV3{}
	eachEntry: // for every entry that exists, find any that has no corresponding model
		for _, entry := range entries {
			if entry.ExternalId != nil {
				_, ok := modelsByExternalID[*entry.ExternalId]
				if ok {
					continue eachEntry // we know the ID and we've found a match, so skip
				}
			}

			// We can't find this entry in our model, or it never had an external ID, which
			// means we want to delete it.
			toDelete = append(toDelete, entry)
		}

		logger.Log("msg", fmt.Sprintf("found %d entries in the catalog, deleting %d of them", len(entries), len(toDelete)))

		// Use a pool of workers to avoid hitting API limits but multiple other
		// routines doing a smash and grab on the rate we do have available.
		if onStart := progress.OnDeleteStart; onStart != nil {
			onStart(len(toDelete))
		}

		p := pool.New().WithErrors().WithContext(ctx).WithMaxGoroutines(10)
		for _, entry := range toDelete {
			var (
				entry = entry // avoid shadow loop variable
			)
			p.Go(func(ctx context.Context) error {
				if onProgress := progress.OnDeleteProgress; onProgress != nil {
					defer onProgress()
				}

				err := cl.Delete(ctx, &entry)
				if err != nil {
					return errors.Wrap(err, "unable to destroy catalog entry, got error")
				}

				logger.Log("msg", "destroyed catalog entry", "catalog_entry_id", entry.Id)
				return nil
			})
		}

		err := p.Wait()
		if err != nil {
			return errors.Wrap(err, "destroying catalog entries")
		}
	}

	// Prepare a quick lookup of entry by external ID. We'll have deleted all entries
	// without an external ID now so can ignore those without one.
	entriesByExternalID := map[string]*client.CatalogEntryV3{}
	for _, entry := range entries {
		if entry.ExternalId != nil {
			entriesByExternalID[*entry.ExternalId] = lo.ToPtr(entry)
		}
	}

	{
		toCreate := []*output.CatalogEntryModel{}
		for _, model := range entryModels {
			_, ok := entriesByExternalID[model.ExternalID]
			if !ok {
				toCreate = append(toCreate, model)
			}
		}

		logger.Log("msg", fmt.Sprintf("found %d entries that need creating", len(toCreate)))

		if onStart := progress.OnCreateStart; onStart != nil {
			onStart(len(toCreate))
		}

		p := pool.New().WithErrors().WithContext(ctx).WithMaxGoroutines(10)
		for _, model := range toCreate {
			var (
				model = model // capture loop variable
			)

			p.Go(func(ctx context.Context) error {
				if onProgress := progress.OnCreateProgress; onProgress != nil {
					defer onProgress()
				}

				result, err := cl.Create(ctx, client.CatalogCreateEntryPayloadV3{
					CatalogTypeId:   catalogType.Id,
					Name:            model.Name,
					Rank:            &model.Rank,
					ExternalId:      lo.ToPtr(model.ExternalID),
					Aliases:         lo.ToPtr(model.Aliases),
					AttributeValues: model.AttributeValues,
				})
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("unable to create catalog entry with external_id=%s, got error", model.ExternalID))
				}

				logger.Log("msg", "created catalog entry", "external_id", model.ExternalID, "entry_id", result.Id)

				return nil
			})
		}

		err := p.Wait()
		if err != nil {
			return errors.Wrap(err, "destroying catalog entries")
		}
	}

	// Identify the attributes that are schema-only, as we want to preserve the existing
	// value instead of setting it outselves.
	attributesToUpdate := []*output.Attribute{}
	for _, attr := range outputType.Attributes {
		if attr.IncludeInPayload() {
			attributesToUpdate = append(attributesToUpdate, attr)
		}
	}

	{
		toUpdate := []client.PartialEntryPayloadV3{}
	eachPayload:
		for _, model := range entryModels {
			entry, ok := entriesByExternalID[model.ExternalID]
			if !ok {
				continue // will have been created above
			}

			// If we found the entry in the list of all entries, then we need to diff it and
			// update as appropriate.
			if entry != nil {
				propsSame :=
					entry.Name == model.Name &&
						reflect.DeepEqual(entry.Aliases, model.Aliases) && entry.Rank == model.Rank

				attributesSame := attributesAreSame(entry.AttributeValues, model.AttributeValues, attributesToUpdate)

				if propsSame && attributesSame {
					logger.Log("msg", "catalog entry has not changed, not updating", "entry_id", entry.Id)
					continue eachPayload
				} else {
					logger.Log("msg", "catalog entry has changed, scheduling for update", "entry_id", entry.Id)

					// Build PartialEntryPayloadV3
					toUpdate = append(toUpdate, client.PartialEntryPayloadV3{
						EntryId:         entry.Id,
						Name:            &model.Name,
						Rank:            &model.Rank,
						ExternalId:      lo.ToPtr(model.ExternalID),
						Aliases:         lo.ToPtr(model.Aliases),
						AttributeValues: model.AttributeValues,
					})
				}
			}
		}

		logger.Log("msg", fmt.Sprintf("found %d entries that need updating", len(toUpdate)))

		if onStart := progress.OnUpdateStart; onStart != nil {
			onStart(len(toUpdate))
		}

		// Chunk into batches of 100
		batches := lo.Chunk(toUpdate, 100)

		// Compute updateAttributes once (same for all batches)
		updateAttributes := lo.ToPtr(lo.Map(attributesToUpdate, func(attr *output.Attribute, _ int) string { return attr.ID }))

		// Process batches SEQUENTIALLY (no pool) to respect rate limits
		// The client's retry logic will handle 429s automatically
		for _, batch := range batches {
			logger.Log("msg", fmt.Sprintf("bulk updating %d catalog entries", len(batch)))

			err := cl.BulkUpdate(ctx, catalogType.Id, batch, updateAttributes)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("unable to bulk update %d catalog entries", len(batch)))
			}

			logger.Log("msg", "bulk updated catalog entries", "count", len(batch))

			// Call progress callback for each entry in the batch
			if onProgress := progress.OnUpdateProgress; onProgress != nil {
				for range batch {
					onProgress()
				}
			}
		}
	}

	return nil
}

// GetEntries paginates through all catalog entries for the given type.
func GetEntries(ctx context.Context, cl *client.ClientWithResponses, catalogTypeID string, pageSize int) (catalogType *client.CatalogTypeV3, entries []client.CatalogEntryV3, err error) {
	var (
		after *string
	)

	for {
		result, err := cl.CatalogV3ListEntriesWithResponse(ctx, &client.CatalogV3ListEntriesParams{
			CatalogTypeId: catalogTypeID,
			PageSize:      int64(pageSize),
			After:         after,
		})
		if err != nil {
			return nil, nil, errors.Wrap(err, "listing entries")
		}

		entries = append(entries, result.JSON200.CatalogEntries...)
		if count := len(result.JSON200.CatalogEntries); count == 0 {
			return &result.JSON200.CatalogType, entries, nil // end pagination
		} else {
			after = lo.ToPtr(result.JSON200.CatalogEntries[count-1].Id)
		}
	}
}

func attributesAreSame(existing map[string]client.CatalogEntryEngineParamBindingV3, desired map[string]client.CatalogEngineParamBindingPayloadV3, attributesToCheck []*output.Attribute) bool {
	// Loop through the attributes which we are in control of and see if any have changed.
	for _, attr := range attributesToCheck {
		if !reflect.DeepEqual(bindingToPayload(existing[attr.ID]), desired[attr.ID]) {
			return false
		}
	}

	return true
}

func bindingToPayload(binding client.CatalogEntryEngineParamBindingV3) client.CatalogEngineParamBindingPayloadV3 {
	payload := client.CatalogEngineParamBindingPayloadV3{}
	if binding.Value != nil {
		payload.Value = &client.CatalogEngineParamBindingValuePayloadV3{
			Literal: binding.Value.Literal,
		}
	}

	if binding.ArrayValue != nil && len(*binding.ArrayValue) > 0 {
		payload.ArrayValue = &[]client.CatalogEngineParamBindingValuePayloadV3{}
		for _, value := range *binding.ArrayValue {
			*payload.ArrayValue = append(*payload.ArrayValue, client.CatalogEngineParamBindingValuePayloadV3{
				Literal: value.Literal,
			})
		}
	}
	return payload
}
