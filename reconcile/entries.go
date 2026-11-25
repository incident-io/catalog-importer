package reconcile

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"reflect"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/output"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/sourcegraph/conc/pool"
)

const (
	bulkCreateBatchSize  = 250
	bulkArchiveBatchSize = 250
	bulkUpdateBatchSize  = 250
)

type EntriesClient struct {
	GetEntries func(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV3, []client.CatalogEntryV3, error)
	Delete     func(ctx context.Context, entry *client.CatalogEntryV3) error
	Create     func(ctx context.Context, payload client.CatalogCreateEntryPayloadV3) (*client.CatalogEntryV3, error)
	BulkUpdate func(ctx context.Context, catalogTypeID string, entries []client.PartialEntryPayloadV3, updateAttributes *[]string) error

	// BulkCreate creates every entry, or none of them, returning the entries in the same order
	// as the payloads. If it is nil, or returns an error that satisfies client.IsNotFound (an
	// API without the endpoint), we fall back to calling Create for each entry.
	BulkCreate func(ctx context.Context, catalogTypeID string, payloads []client.CatalogCreateEntryPayloadV3) ([]client.CatalogEntryV3, error)
	// BulkArchive archives every entry, or none of them, skipping any that are already gone.
	// If it is nil, or returns an error that satisfies client.IsNotFound, we fall back to
	// calling Delete for each entry.
	BulkArchive func(ctx context.Context, catalogTypeID string, entries []client.CatalogEntryV3) error
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
		BulkCreate: func(ctx context.Context, catalogTypeID string, payloads []client.CatalogCreateEntryPayloadV3) ([]client.CatalogEntryV3, error) {
			result, err := cl.CatalogV3BulkCreateEntriesWithResponse(ctx, client.CatalogBulkCreateEntriesPayloadV3{
				CatalogTypeId: catalogTypeID,
				Entries: lo.Map(payloads, func(payload client.CatalogCreateEntryPayloadV3, _ int) client.BulkCreateEntryPayloadV3 {
					return client.BulkCreateEntryPayloadV3{
						ExternalId:      lo.FromPtr(payload.ExternalId),
						Name:            payload.Name,
						Rank:            payload.Rank,
						Aliases:         payload.Aliases,
						AttributeValues: payload.AttributeValues,
					}
				}),
			})
			if err != nil {
				return nil, err
			}
			if result.JSON201 == nil {
				return nil, errors.Errorf("unexpected nil 201 response. Status Code: %d", result.HTTPResponse.StatusCode)
			}

			return result.JSON201.CatalogEntries, nil
		},
		BulkArchive: func(ctx context.Context, catalogTypeID string, entries []client.CatalogEntryV3) error {
			result, err := cl.CatalogV3BulkArchiveEntriesWithResponse(ctx, client.CatalogBulkArchiveEntriesPayloadV3{
				CatalogTypeId: catalogTypeID,
				EntryIds:      lo.Map(entries, func(entry client.CatalogEntryV3, _ int) string { return entry.Id }),
			})
			if err != nil {
				return err
			}
			if result.StatusCode() != http.StatusNoContent {
				return errors.Errorf("unexpected response. Status Code: %d", result.StatusCode())
			}

			return nil
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

		if onStart := progress.OnDeleteStart; onStart != nil {
			onStart(len(toDelete))
		}

		err := deleteEntries(ctx, logger, cl, catalogType.Id, toDelete, progress.OnDeleteProgress)
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

		payloads := lo.Map(toCreate, func(model *output.CatalogEntryModel, _ int) client.CatalogCreateEntryPayloadV3 {
			return client.CatalogCreateEntryPayloadV3{
				CatalogTypeId:   catalogType.Id,
				Name:            model.Name,
				Rank:            lo.ToPtr(model.Rank),
				ExternalId:      lo.ToPtr(model.ExternalID),
				Aliases:         lo.ToPtr(model.Aliases),
				AttributeValues: model.AttributeValues,
			}
		})

		err := createEntries(ctx, logger, cl, catalogType.Id, payloads, progress.OnCreateProgress)
		if err != nil {
			return errors.Wrap(err, "creating catalog entries")
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

		batches := lo.Chunk(toUpdate, bulkUpdateBatchSize)

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

// deleteEntries archives entries in batches through BulkArchive, falling back to one
// Delete per entry if the API doesn't support bulk archiving.
func deleteEntries(ctx context.Context, logger kitlog.Logger, cl EntriesClient, catalogTypeID string, entries []client.CatalogEntryV3, onProgress func()) error {
	if cl.BulkArchive == nil {
		return deleteEntriesOneByOne(ctx, logger, cl, entries, onProgress)
	}

	errs := []error{}
	for start := 0; start < len(entries); start += bulkArchiveBatchSize {
		batch := entries[start:min(start+bulkArchiveBatchSize, len(entries))]

		logger.Log("msg", fmt.Sprintf("bulk archiving %d catalog entries", len(batch)))
		err := cl.BulkArchive(ctx, catalogTypeID, batch)
		if client.IsNotFound(err) {
			logger.Log("msg", "bulk archive endpoint not found, falling back to deleting entries one at a time")
			errs = append(errs, deleteEntriesOneByOne(ctx, logger, cl, entries[start:], onProgress))
			break
		}
		// The whole batch fails if any entry sets an attribute we can't clear, so retry it
		// one entry at a time to archive the others and report the ones we can't.
		if client.HasErrorCode(err, "restricted_attribute") {
			logger.Log("msg", "bulk archive includes entries with restricted attributes, deleting this batch one at a time")
			errs = append(errs, deleteEntriesOneByOne(ctx, logger, cl, batch, onProgress))
			continue
		}

		for _, entry := range batch {
			if err != nil {
				errs = append(errs, errors.Wrap(err, fmt.Sprintf("unable to destroy catalog entry with id=%s, got error", entry.Id)))
			} else {
				logger.Log("msg", "destroyed catalog entry", "catalog_entry_id", entry.Id)
			}

			if onProgress != nil {
				onProgress()
			}
		}
	}

	return stderrors.Join(errs...)
}

func deleteEntriesOneByOne(ctx context.Context, logger kitlog.Logger, cl EntriesClient, entries []client.CatalogEntryV3, onProgress func()) error {
	// Use a pool of workers to avoid hitting API limits but multiple other
	// routines doing a smash and grab on the rate we do have available.
	p := pool.New().WithErrors().WithContext(ctx).WithMaxGoroutines(10)
	for _, entry := range entries {
		p.Go(func(ctx context.Context) error {
			if onProgress != nil {
				defer onProgress()
			}

			err := cl.Delete(ctx, &entry)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("unable to destroy catalog entry with id=%s, got error", entry.Id))
			}

			logger.Log("msg", "destroyed catalog entry", "catalog_entry_id", entry.Id)
			return nil
		})
	}

	return p.Wait()
}

// createEntries creates entries in batches through BulkCreate, falling back to one Create
// per entry if the API doesn't support bulk creating.
func createEntries(ctx context.Context, logger kitlog.Logger, cl EntriesClient, catalogTypeID string, payloads []client.CatalogCreateEntryPayloadV3, onProgress func()) error {
	if cl.BulkCreate == nil {
		return createEntriesOneByOne(ctx, logger, cl, payloads, onProgress)
	}

	errs := []error{}
	for start := 0; start < len(payloads); start += bulkCreateBatchSize {
		batch := payloads[start:min(start+bulkCreateBatchSize, len(payloads))]

		logger.Log("msg", fmt.Sprintf("bulk creating %d catalog entries", len(batch)))
		entries, err := cl.BulkCreate(ctx, catalogTypeID, batch)
		if client.IsNotFound(err) {
			logger.Log("msg", "bulk create endpoint not found, falling back to creating entries one at a time")
			errs = append(errs, createEntriesOneByOne(ctx, logger, cl, payloads[start:], onProgress))
			break
		}
		// A bulk create is all or nothing, so one invalid entry fails the whole batch. Create
		// this batch one entry at a time instead, to create the others and report that entry.
		if client.HasStatus(err, http.StatusUnprocessableEntity) {
			logger.Log("msg", "bulk create failed validation, creating this batch one entry at a time", "error", err)
			errs = append(errs, createEntriesOneByOne(ctx, logger, cl, batch, onProgress))
			continue
		}
		if err == nil && len(entries) != len(batch) {
			err = errors.Errorf("expected %d created entries, got %d", len(batch), len(entries))
		}

		for idx, payload := range batch {
			externalID := lo.FromPtr(payload.ExternalId)
			if err != nil {
				errs = append(errs, errors.Wrap(err, fmt.Sprintf("unable to create catalog entry with external_id=%s, got error", externalID)))
			} else {
				logger.Log("msg", "created catalog entry", "external_id", externalID, "entry_id", entries[idx].Id)
			}

			if onProgress != nil {
				onProgress()
			}
		}
	}

	return stderrors.Join(errs...)
}

func createEntriesOneByOne(ctx context.Context, logger kitlog.Logger, cl EntriesClient, payloads []client.CatalogCreateEntryPayloadV3, onProgress func()) error {
	p := pool.New().WithErrors().WithContext(ctx).WithMaxGoroutines(10)
	for _, payload := range payloads {
		p.Go(func(ctx context.Context) error {
			if onProgress != nil {
				defer onProgress()
			}

			externalID := lo.FromPtr(payload.ExternalId)
			result, err := cl.Create(ctx, payload)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("unable to create catalog entry with external_id=%s, got error", externalID))
			}

			logger.Log("msg", "created catalog entry", "external_id", externalID, "entry_id", result.Id)
			return nil
		})
	}

	return p.Wait()
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
