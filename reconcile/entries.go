package reconcile

import (
	"context"
	"fmt"
	"reflect"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/client"
	"github.com/incident-io/catalog-importer/output"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

type EntriesClient struct {
	GetEntries func(ctx context.Context, catalogTypeID string) (*client.CatalogTypeV2, []client.CatalogEntryV2, error)
	Delete     func(ctx context.Context, entry *client.CatalogEntryV2) error
	Create     func(ctx context.Context, payload client.CreateEntryRequestBody) (*client.CatalogEntryV2, error)
	Update     func(ctx context.Context, entry *client.CatalogEntryV2, payload client.UpdateEntryRequestBody) (*client.CatalogEntryV2, error)
}

// EntriesClientFromClient wraps a real client with hooks that can create, update and delete
// entries. This can be overriden for custom behaviour, such as a dry-run that shouldn't
// actually perform updates.
func EntriesClientFromClient(cl *client.ClientWithResponses) EntriesClient {
	return EntriesClient{
		GetEntries: func(ctx context.Context, catalogTypeID string) (*client.CatalogTypeV2, []client.CatalogEntryV2, error) {
			return GetEntries(ctx, cl, catalogTypeID)
		},
		Delete: func(ctx context.Context, entry *client.CatalogEntryV2) error {
			_, err := cl.CatalogV2DestroyEntryWithResponse(ctx, entry.Id)
			if err != nil {
				return err
			}

			return nil
		},
		Create: func(ctx context.Context, payload client.CreateEntryRequestBody) (*client.CatalogEntryV2, error) {
			result, err := cl.CatalogV2CreateEntryWithResponse(ctx, payload)
			if err != nil {
				return nil, err
			}

			return &result.JSON201.CatalogEntry, nil
		},
		Update: func(ctx context.Context, entry *client.CatalogEntryV2, payload client.UpdateEntryRequestBody) (*client.CatalogEntryV2, error) {
			result, err := cl.CatalogV2UpdateEntryWithResponse(ctx, entry.Id, payload)
			if err != nil {
				return nil, err
			}

			return &result.JSON200.CatalogEntry, err
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

func Entries(ctx context.Context, logger kitlog.Logger, cl EntriesClient, catalogType *client.CatalogTypeV2, entryModels []*output.CatalogEntryModel, progress *EntriesProgress) error {
	logger = kitlog.With(logger,
		"catalog_type_id", catalogType.Id,
		"catalog_type_name", catalogType.TypeName,
	)

	// Initialise this as it's easy to deal with if you don't nil check the full struct.
	if progress == nil {
		progress = new(EntriesProgress)
	}

	logger.Log("msg", "listing existing entries")
	catalogType, entries, err := cl.GetEntries(ctx, catalogType.Id)
	if err != nil {
		return errors.Wrap(err, "listing entries")
	}

	// Prepare a quick lookup of model by external ID, to power deletion.
	modelsByExternalID := map[string]*output.CatalogEntryModel{}
	for _, model := range entryModels {
		modelsByExternalID[model.ExternalID] = model
	}

	{
		toDelete := []client.CatalogEntryV2{}
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

		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(10)

		if onStart := progress.OnDeleteStart; onStart != nil {
			onStart(len(toDelete))
		}

		for _, entry := range toDelete {
			var (
				entry = entry // avoid shadow loop variable
			)
			g.Go(func() error {
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

		if err := g.Wait(); err != nil {
			return errors.Wrap(err, "destroying catalog entries")
		}
	}

	// Prepare a quick lookup of entry by external ID. We'll have deleted all entries
	// without an external ID now so can ignore those without one.
	entriesByExternalID := map[string]*client.CatalogEntryV2{}
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

		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(10)

		if onStart := progress.OnCreateStart; onStart != nil {
			onStart(len(toCreate))
		}

		for _, model := range toCreate {
			var (
				model = model // capture loop variable
			)

			g.Go(func() error {
				if onProgress := progress.OnCreateProgress; onProgress != nil {
					defer onProgress()
				}

				result, err := cl.Create(ctx, client.CreateEntryRequestBody{
					CatalogTypeId:   catalogType.Id,
					Name:            model.Name,
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

		if err := g.Wait(); err != nil {
			return errors.Wrap(err, "creating new catalog entries")
		}
	}

	{
		toUpdate := []*output.CatalogEntryModel{}
	eachPayload:
		for _, model := range entryModels {
			entry, ok := entriesByExternalID[model.ExternalID]
			if !ok {
				continue // will have been created above
			}

			entry, alreadyExists := entriesByExternalID[model.ExternalID]
			if alreadyExists {
				// If we found the entry in the list of all entries, then we need to diff it and
				// update as appropriate.
				if entry != nil {
					isSame :=
						entry.Name == model.Name &&
							reflect.DeepEqual(entry.Aliases, model.Aliases)

					currentBindings := map[string]client.CatalogAttributeBindingPayloadV2{}
					for attributeID, value := range entry.AttributeValues {
						current := client.CatalogAttributeBindingPayloadV2{}
						// Our API behaves strangely with empty arrays, and will omit them. This patch
						// ensures the array is present so our comparison doesn't trigger falsly.
						if value.ArrayValue == nil && value.Value == nil {
							value.ArrayValue = lo.ToPtr([]client.CatalogAttributeValueV2{})
						}

						if value.ArrayValue != nil {
							current.ArrayValue = lo.ToPtr(lo.Map(*value.ArrayValue, func(binding client.CatalogAttributeValueV2, _ int) client.CatalogAttributeValuePayloadV2 {
								return client.CatalogAttributeValuePayloadV2{
									Literal: binding.Literal,
								}
							}))
						}
						if value.Value != nil {
							current.Value = &client.CatalogAttributeValuePayloadV2{
								Literal: value.Value.Literal,
							}
						}

						currentBindings[attributeID] = current
					}

					if isSame && reflect.DeepEqual(model.AttributeValues, currentBindings) {
						logger.Log("msg", "catalog entry has not changed, not updating", "entry_id", entry.Id)
						continue eachPayload
					} else {
						logger.Log("msg", "catalog entry has changed, scheduling for update", "entry_id", entry.Id)
						toUpdate = append(toUpdate, model)
					}
				}
			}
		}

		logger.Log("msg", fmt.Sprintf("found %d entries that need updated", len(toUpdate)))

		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(10)

		if onStart := progress.OnUpdateStart; onStart != nil {
			onStart(len(toUpdate))
		}

		for _, model := range toUpdate {
			var (
				model = model                                 // capture loop variable
				entry = entriesByExternalID[model.ExternalID] // for ID
			)

			g.Go(func() error {
				if onProgress := progress.OnUpdateProgress; onProgress != nil {
					defer onProgress()
				}

				_, err := cl.Update(ctx, entry, client.UpdateEntryRequestBody{
					Name:            model.Name,
					ExternalId:      lo.ToPtr(model.ExternalID),
					Aliases:         lo.ToPtr(model.Aliases),
					AttributeValues: model.AttributeValues,
				})
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("unable to update catalog entry with id=%s, got error", entry.Id))
				}

				logger.Log("msg", "updated catalog entry", "entry_id", entry.Id)
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return errors.Wrap(err, "updating catalog entries")
		}
	}

	return nil
}

// GetEntries paginates through all catalog entries for the given type.
func GetEntries(ctx context.Context, cl *client.ClientWithResponses, catalogTypeID string) (catalogType *client.CatalogTypeV2, entries []client.CatalogEntryV2, err error) {
	var (
		after *string
	)

	for {
		result, err := cl.CatalogV2ListEntriesWithResponse(ctx, &client.CatalogV2ListEntriesParams{
			CatalogTypeId: catalogTypeID,
			PageSize:      lo.ToPtr(int(250)),
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
