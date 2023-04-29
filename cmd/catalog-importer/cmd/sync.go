package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/incident-io/catalog-importer/client"
	"github.com/incident-io/catalog-importer/output"
	"github.com/incident-io/catalog-importer/reconcile"
	"github.com/incident-io/catalog-importer/source"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"
)

type SyncOptions struct {
	ConfigFile  string
	APIEndpoint string
	APIKey      string
	DryRun      bool
	Prune       bool
}

func (opt *SyncOptions) Bind(cmd *kingpin.CmdClause) *SyncOptions {
	cmd.Flag("config", "Config file in either Jsonnet, YAML or JSON (e.g. config.jsonnet)").
		StringVar(&opt.ConfigFile)
	cmd.Flag("api-endpoint", "Endpoint of the incident.io API").
		Default("https://api.incident.io").
		Envar("INCIDENT_ENDPOINT").
		StringVar(&opt.APIEndpoint)
	cmd.Flag("api-key", "API key for incident.io").
		Envar("INCIDENT_API_KEY").
		StringVar(&opt.APIKey)
	cmd.Flag("dry-run", "Only calculate the changes needed and print the diff, don't actually make changes").
		Default("false").
		BoolVar(&opt.DryRun)
	cmd.Flag("prune", "Remove catalog types that are no longer in the config").
		BoolVar(&opt.Prune)

	return opt
}

func (opt *SyncOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	if opt.DryRun && opt.Prune {
		return errors.New("cannot use --dry-run with --prune")
	}

	// Load config
	cfg, err := loadConfigOrError(ctx, opt.ConfigFile)
	if err != nil {
		return err
	}
	{
		var (
			outputs, sources int
		)
		for _, pipeline := range cfg.Pipelines {
			outputs += len(pipeline.Outputs)
			sources += len(pipeline.Sources)
		}
		OUT("✔ Loaded config (%d pipelines, %d sources, %d outputs)", len(cfg.Pipelines), outputs, sources)
	}

	clientOptions := []client.ClientOption{}
	if opt.DryRun {
		OUT("⛨ --dry-run is set, building a read-only client")
		clientOptions = append(clientOptions, client.WithReadOnly())
	}

	// Build incident.io client
	cl, err := client.New(ctx, opt.APIKey, opt.APIEndpoint, Version(), clientOptions...)
	if err != nil {
		return err
	}
	OUT("✔ Connected to incident.io API (%s)", opt.APIEndpoint)

	// Load existing catalog types
	result, err := cl.CatalogV2ListTypesWithResponse(ctx)
	if err != nil {
		return errors.Wrap(err, "listing catalog types")
	}

	existingCatalogTypes := []client.CatalogTypeV2{}
	for _, catalogType := range result.JSON200.CatalogTypes {
		logger := kitlog.With(logger,
			"catalog_type_id", catalogType.Id,
			"catalog_type_name", catalogType.Name,
		)

		syncID, ok := catalogType.Annotations[AnnotationSyncID]
		if !ok {
			level.Debug(logger).Log("msg", "ignoring catalog type as it is not managed by an importer")
		} else if syncID != cfg.SyncID {
			logger.Log("msg", "ignoring catalog type as it is managed by a different importer",
				"catalog_type_sync_id", syncID)
		} else {
			existingCatalogTypes = append(existingCatalogTypes, catalogType)
		}
	}

	logger.Log("msg", "found managed catalog types",
		"catalog_types", strings.Join(lo.Map(existingCatalogTypes, func(ct client.CatalogTypeV2, _ int) string {
			return ct.TypeName
		}), ", "))
	OUT("✔ Found %d catalog types, with %d that match our sync ID (%s)",
		len(result.JSON200.CatalogTypes), len(existingCatalogTypes), cfg.SyncID)

	// Remove unmanaged types
	if opt.Prune {
		OUT("\n↻ Prune enabled (--prune), removing types that are no longer in config...")

		toDestroy := []client.CatalogTypeV2{}
	nextCatalogType:
		for _, existingCatalogType := range existingCatalogTypes {
			logger := kitlog.With(logger,
				"type_name", existingCatalogType.TypeName,
				"catalog_type_id", existingCatalogType.Id,
			)

			for _, output := range cfg.Outputs() {
				if output.TypeName == existingCatalogType.TypeName {
					level.Debug(logger).Log("catalog type already exists")
					continue nextCatalogType
				}
			}

			toDestroy = append(toDestroy, existingCatalogType)
		}

		if len(toDestroy) == 0 {
			OUT("  ✔ Nothing to remove!")
		} else {
			for _, catalogType := range toDestroy {
				logger.Log("msg", "found catalog type for this sync ID that is no longer in config, removing")
				_, err := cl.CatalogV2DestroyTypeWithResponse(ctx, catalogType.Id)
				if err != nil {
					return errors.Wrap(err, "removing catalog type")
				}
				OUT("  ⌫ %s", catalogType.TypeName)
			}
		}
	}

	// Create missing catalog types
	OUT("\n↻ Creating catalog types that don't yet exist...")
createCatalogType:
	for _, outputType := range cfg.Outputs() {
		logger := kitlog.With(logger, "type_name", outputType.TypeName)

		model := output.MarshalType(outputType)
		for _, existingCatalogType := range existingCatalogTypes {
			if model.TypeName == existingCatalogType.TypeName {
				level.Debug(logger).Log("catalog type already exists")
				continue createCatalogType
			}
		}

		var createdCatalogType client.CatalogTypeV2
		if opt.DryRun {
			logger.Log("msg", "catalog type does not already exist, simulating create for --dry-run")
			createdCatalogType = client.CatalogTypeV2{
				Id:          fmt.Sprintf("DRY-RUN-%s", model.TypeName),
				Name:        model.Name,
				Description: model.Description,
				TypeName:    model.TypeName,
			}
		} else {
			logger.Log("msg", "catalog type does not already exist, creating")
			result, err := cl.CatalogV2CreateTypeWithResponse(ctx, client.CreateTypeRequestBody{
				Name:        model.Name,
				Description: model.Description,
				TypeName:    lo.ToPtr(model.TypeName),
				Annotations: lo.ToPtr(getAnnotations(cfg.SyncID)),
			})
			if err != nil {
				return errors.Wrap(err, "creating catalog type")
			}

			createdCatalogType = result.JSON201.CatalogType
			logger.Log("msg", "created catalog type", "catalog_type_id", createdCatalogType.Id)
		}

		existingCatalogTypes = append(existingCatalogTypes, createdCatalogType)
		OUT("  ✔ %s (id=%s)", outputType.TypeName, createdCatalogType)
	}

	// Prepare a lookup of catalog type by the output name for subsequent pipeline steps.
	catalogTypesByOutput := map[string]*client.CatalogTypeV2{}
	for _, outputType := range cfg.Outputs() {
		var catalogType *client.CatalogTypeV2
		for _, existingCatalogType := range existingCatalogTypes {
			if outputType.TypeName == existingCatalogType.TypeName {
				catalogType = &existingCatalogType
				break
			}
		}
		if catalogType == nil {
			return fmt.Errorf("could not find catalog type for model '%s', this is a bug in the importer", outputType.TypeName)
		}

		catalogTypesByOutput[outputType.TypeName] = catalogType
	}

	// Update type schemas to match config
	OUT("\n↻ Syncing catalog type schemas...")
	for _, outputType := range cfg.Outputs() {
		var (
			model       = output.MarshalType(outputType)
			catalogType = catalogTypesByOutput[outputType.TypeName]
		)

		var updatedCatalogType client.CatalogTypeV2
		if opt.DryRun {
			logger.Log("msg", "dry-run active, which means we fake a response")
			updatedCatalogType = *catalogType // they start the same

			// Then we pretend like we've already updated the schema, which means we rebuild the
			// attributes.
			updatedCatalogType.Schema = client.CatalogTypeSchemaV2{
				Version:    updatedCatalogType.Schema.Version,
				Attributes: []client.CatalogTypeAttributeV2{},
			}
			for _, attr := range model.Attributes {
				updatedCatalogType.Schema.Attributes = append(updatedCatalogType.Schema.Attributes, client.CatalogTypeAttributeV2{
					Id:    *attr.Id,
					Name:  attr.Name,
					Type:  attr.Type,
					Array: attr.Array,
				})
			}
		} else {
			logger.Log("msg", "updating catalog type", "catalog_type_id", catalogType.Id)
			result, err := cl.CatalogV2UpdateTypeWithResponse(ctx, catalogType.Id, client.CatalogV2UpdateTypeJSONRequestBody{
				Name:        model.Name,
				Description: model.Description,
				TypeName:    lo.ToPtr(model.TypeName),
				Annotations: lo.ToPtr(getAnnotations(cfg.SyncID)),
			})
			if err != nil {
				return errors.Wrap(err, "updating catalog type")
			}

			version := result.JSON200.CatalogType.Schema.Version
			logger.Log("msg", "updating catalog type schema", "catalog_type_id", catalogType.Id, "version", version)
			schemaResult, err := cl.CatalogV2UpdateTypeSchemaWithResponse(ctx, catalogType.Id, client.CatalogV2UpdateTypeSchemaJSONRequestBody{
				Version:    version,
				Attributes: model.Attributes,
			})
			if err != nil {
				return errors.Wrap(err, "updating catalog type schema")
			}

			updatedCatalogType = schemaResult.JSON200.CatalogType
		}
		OUT("  ✔ %s (id=%s)", outputType.TypeName, catalogType.Id)
		if opt.DryRun {
			DIFF("  ", *catalogType, updatedCatalogType)
		}
	}

	for _, pipeline := range cfg.Pipelines {
		OUT("\n↻ Syncing pipeline... (%s)", strings.Join(lo.Map(pipeline.Outputs, func(op *output.Output, _ int) string {
			return op.TypeName
		}), ", "))

		// Load entries from source
		sourcedEntries := []source.Entry{}
		{
			OUT("\n  ↻ Loading data from sources...")
			for _, source := range pipeline.Sources {
				sourceEntries, err := source.Load(ctx)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("loading entries from source %v", source))
				}

				for _, sourceEntry := range sourceEntries {
					parsedEntries, err := sourceEntry.Parse()
					if err != nil {
						return errors.Wrap(err, fmt.Sprintf("parsing source entry: %s", sourceEntry.Origin))
					}

					sourcedEntries = append(sourcedEntries, parsedEntries...)
				}

				OUT("    ✔ %s (found %d entries)", source.Name(), len(sourcedEntries))
			}
		}

		OUT("\n  ↻ Syncing entries...")
		for idx, outputType := range pipeline.Outputs {
			OUT("\n    ↻ %s", outputType.TypeName)

			// Filter source for each of the output types
			entries, err := output.Collect(ctx, outputType, sourcedEntries)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("outputs.%d (type_name='%s')", idx, outputType.TypeName))
			}
			OUT("      ✔ Building entries... (found %d entries matching filters)", len(entries))

			// Marshal entries using the CEL expressions.
			entryModels, err := output.MarshalEntries(ctx, outputType, entries)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("outputs.%d (type_name='%s')", idx, outputType.TypeName))
			}

			logger.Log("msg", "reconciling catalog entries", "output", outputType.TypeName)
			catalogType := catalogTypesByOutput[outputType.TypeName]

			err = reconcile.Entries(ctx, logger, newEntriesClient(cl, existingCatalogTypes, opt.DryRun), catalogType, entryModels, newEntriesProgress(!opt.DryRun))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("outputs (type_name = '%s'): reconciling catalog entries", outputType.TypeName))
			}
		}
	}

	return nil
}

// newEntriesClient will return a client that speaks to the real API if dry-run is false,
// or we'll create a no-op client that just outputs diffs.
func newEntriesClient(cl *client.ClientWithResponses, existingCatalogTypes []client.CatalogTypeV2, dryRun bool) reconcile.EntriesClient {
	if !dryRun {
		return reconcile.EntriesClientFromClient(cl)
	}

	return reconcile.EntriesClient{
		GetEntries: func(ctx context.Context, catalogTypeID string) (*client.CatalogTypeV2, []client.CatalogEntryV2, error) {
			// We're in dry-run and this catalog type is yet to be created. We can't ask the API
			// for the entries of a type that doesn't exist, so we return the type we faked from
			// the dry-run create and an empty list of entries.
			if strings.HasPrefix(catalogTypeID, "DRY-RUN") {
				for _, existingType := range existingCatalogTypes {
					if existingType.Id == catalogTypeID {
						return &existingType, []client.CatalogEntryV2{}, nil
					}
				}

				return nil, nil, fmt.Errorf("could not find dry-run faked catalog type with id='%s'", catalogTypeID)
			}

			// We're just a normal catalog type, use the real client.
			return reconcile.GetEntries(ctx, cl, catalogTypeID)
		},
		Delete: func(ctx context.Context, entry *client.CatalogEntryV2) error {
			DIFF("      ", *entry, client.CatalogEntryV2{})
			return nil
		},
		Create: func(ctx context.Context, payload client.CreateEntryRequestBody) (*client.CatalogEntryV2, error) {
			DIFF("      ", client.CreateEntryRequestBody{}, payload)
			entry := &client.CatalogEntryV2{
				Id: fmt.Sprintf("DRY-RUN-%s", uuid.NewString()),
			}

			return entry, nil
		},
		Update: func(ctx context.Context, entry *client.CatalogEntryV2, payload client.UpdateEntryRequestBody) (*client.CatalogEntryV2, error) {
			existingPayload := client.UpdateEntryRequestBody{
				Aliases:         lo.ToPtr(entry.Aliases),
				AttributeValues: map[string]client.CatalogAttributeBindingPayloadV2{},
				ExternalId:      entry.ExternalId,
				Name:            entry.Name,
				Rank:            &entry.Rank,
			}
			if payload.Rank == nil && entry.Rank == 0 {
				existingPayload.Rank = nil
			}
			for attrID, attr := range entry.AttributeValues {
				result := client.CatalogAttributeBindingPayloadV2{}
				if attr.Value != nil {
					result.Value = &client.CatalogAttributeValuePayloadV2{
						Literal: attr.Value.Literal,
					}
				}
				if attr.ArrayValue != nil {
					arrayValue := []client.CatalogAttributeValuePayloadV2{}
					for _, elementValue := range *attr.ArrayValue {
						arrayValue = append(arrayValue, client.CatalogAttributeValuePayloadV2{
							Literal: elementValue.Literal,
						})
					}

					result.ArrayValue = &arrayValue
				}

				existingPayload.AttributeValues[attrID] = result
			}

			DIFF("      ", existingPayload, payload)
			return entry, nil
		},
	}
}

// newEntriesProgress creates hooks to render progress into the terminal while reconciling
// catalog entries.
func newEntriesProgress(showBars bool) *reconcile.EntriesProgress {
	var (
		deleteBar *progressbar.ProgressBar
		createBar *progressbar.ProgressBar
		updateBar *progressbar.ProgressBar
	)
	return &reconcile.EntriesProgress{
		OnDeleteStart: func(total int) {
			if total == 0 {
				OUT("      ✔ No entries to delete")
			} else {
				OUT("      ✔ Deleting unmanaged entries... (found %d entries in catalog not in source)", total)
				if showBars {
					deleteBar = newProgressBar(int64(total),
						progressbar.OptionSetDescription(`        `),
					)
				}
			}
		},
		OnDeleteProgress: func() {
			if deleteBar != nil {
				deleteBar.Add(1)
			}
		},
		OnCreateStart: func(total int) {
			if total == 0 {
				OUT("      ✔ No new entries to create")
			} else {
				OUT("      ✔ Creating new entries in catalog... (%d entries to create)", total)
				if showBars {
					createBar = newProgressBar(int64(total),
						progressbar.OptionSetDescription(`        `),
					)
				}
			}
		},
		OnCreateProgress: func() {
			if createBar != nil {
				createBar.Add(1)
			}
		},
		OnUpdateStart: func(total int) {
			if total == 0 {
				OUT("      ✔ No existing entries to update")
			} else {
				OUT("      ✔ Updating existing entries in catalog... (%d entries to update)", total)
				if showBars {
					updateBar = newProgressBar(int64(total),
						progressbar.OptionSetDescription(`        `),
					)
				}
			}
		},
		OnUpdateProgress: func() {
			if updateBar != nil {
				updateBar.Add(1)
			}
		},
	}
}

var (
	AnnotationSyncID     = "incident.io/catalog-importer/sync-id"
	AnnotationLastSyncAt = "incident.io/catalog-importer/last-sync-at"
	AnnotationVersion    = "incident.io/catalog-importer/version"
)

func getAnnotations(syncID string) map[string]string {
	return map[string]string{
		AnnotationSyncID:     syncID,
		AnnotationLastSyncAt: time.Now().Format(time.RFC3339),
		AnnotationVersion:    Version(),
	}
}

func newProgressBar(total int64, opts ...progressbar.Option) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		total,
		append([]progressbar.Option{
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionSetWidth(40),
			progressbar.OptionThrottle(65 * time.Millisecond),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprint(os.Stderr, "\n")
			}),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionSetRenderBlankState(true),
		}, opts...,
		)...,
	)
}
