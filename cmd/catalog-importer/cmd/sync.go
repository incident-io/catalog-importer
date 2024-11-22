package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"

	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/config"
	"github.com/incident-io/catalog-importer/v2/output"
	"github.com/incident-io/catalog-importer/v2/reconcile"
	"github.com/incident-io/catalog-importer/v2/source"
)

type SyncOptions struct {
	ConfigFile                string
	APIEndpoint               string
	APIKey                    string
	Targets                   []string
	SampleLength              int
	DryRun                    bool
	Prune                     bool
	AllowDeleteAll            bool
	SourceRepoUrl             string
	CatalogEntriesAPIPageSize int
}

func (opt *SyncOptions) Bind(cmd *kingpin.CmdClause) *SyncOptions {
	cmd.Flag("config", "Config file in either Jsonnet, YAML or JSON (e.g. importer.jsonnet)").
		StringVar(&opt.ConfigFile)
	cmd.Flag("api-endpoint", "Endpoint of the incident.io API").
		Default("https://api.incident.io").
		Envar("INCIDENT_ENDPOINT").
		StringVar(&opt.APIEndpoint)
	cmd.Flag("api-key", "API key for incident.io").
		Envar("INCIDENT_API_KEY").
		StringVar(&opt.APIKey)
	cmd.Flag("source-repo-url", "URL of repo where catalog is being managed").
		Envar("SOURCE_REPO_URL").
		StringVar(&opt.SourceRepoUrl)
	cmd.Flag("target", `Restrict running to only these outputs (e.g. Custom["Customer"])`).
		StringsVar(&opt.Targets)
	cmd.Flag("sample-length", "How many character to sample when logging about invalid source entries (for --debug only)").
		Default("256").
		IntVar(&opt.SampleLength)
	cmd.Flag("dry-run", "Only calculate the changes needed and print the diff, don't actually make changes").
		Default("false").
		BoolVar(&opt.DryRun)
	cmd.Flag("prune", "Remove catalog types that are no longer in the config").
		BoolVar(&opt.Prune)
	cmd.Flag("allow-delete-all", "Allow removing all entries from a catalog entry").
		BoolVar(&opt.AllowDeleteAll)
	cmd.Flag("catalog-entries-api-page-size", "The page size to use when listing catalog entries from the API").
		Envar("CATALOG_ENTRIES_API_PAGE_SIZE").
		Default("250").
		IntVar(&opt.CatalogEntriesAPIPageSize)

	return opt
}

func (opt *SyncOptions) Run(ctx context.Context, logger kitlog.Logger, cfg *config.Config) error {
	if opt.Prune && opt.DryRun {
		return errors.New("cannot use --dry-run with --prune")
	}
	if opt.Prune && len(opt.Targets) > 0 {
		return errors.New("cannot use --targets with --prune")
	}

	// Load config if it hasn't been provided.
	if cfg == nil {
		var err error
		cfg, err = loadConfigOrError(ctx, opt.ConfigFile)
		if err != nil {
			return err
		}
	}
	{
		if len(opt.Targets) > 0 {
			OUT("⊕ Filtering config to targets (%s)", strings.Join(opt.Targets, ", "))
			cfg = cfg.Filter(opt.Targets)
		}
		var outputs, sources int
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
	cl, err := client.New(ctx, opt.APIKey, opt.APIEndpoint, Version(), logger, clientOptions...)
	if err != nil {
		return err
	}

	// Load existing catalog types
	result, err := cl.CatalogV2ListTypesWithResponse(ctx)
	if err != nil {
		return errors.Wrap(err, "listing catalog types")
	}
	OUT("✔ Connected to incident.io API (%s)", opt.APIEndpoint)

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
	for _, outputType := range cfg.Outputs() {
		logger := kitlog.With(logger, "type_name", outputType.TypeName)

		baseModel, enumModels := output.MarshalType(outputType)
	createCatalogType:
		for _, model := range append(enumModels, baseModel) {
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
					Id:            fmt.Sprintf("DRY-RUN-%s", model.TypeName),
					Name:          model.Name,
					Description:   model.Description,
					TypeName:      model.TypeName,
					SourceRepoUrl: &opt.SourceRepoUrl,
				}
			} else {
				logger.Log("msg", "catalog type does not already exist, creating")
				categories := lo.Map(model.Categories, func(category string, _ int) client.CreateTypeRequestBodyCategories {
					return client.CreateTypeRequestBodyCategories(category)
				})

				result, err := cl.CatalogV2CreateTypeWithResponse(ctx, client.CreateTypeRequestBody{
					Name:          model.Name,
					Description:   model.Description,
					Ranked:        &model.Ranked,
					TypeName:      lo.ToPtr(model.TypeName),
					Categories:    lo.ToPtr(categories),
					Annotations:   lo.ToPtr(getAnnotations(cfg.SyncID)),
					SourceRepoUrl: &opt.SourceRepoUrl,
				})
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("creating catalog type with name %s", model.TypeName))
				}

				createdCatalogType = result.JSON201.CatalogType
				logger.Log("msg", "created catalog type", "catalog_type_id", createdCatalogType.Id)
			}

			existingCatalogTypes = append(existingCatalogTypes, createdCatalogType)
			OUT("  ✔ %s (id=%s)", model.TypeName, createdCatalogType.Id)
		}
	}

	// Prepare a lookup of catalog type by the output name for subsequent pipeline steps.
	catalogTypesByOutput := map[string]*client.CatalogTypeV2{}
	for _, outputType := range cfg.Outputs() {
		baseModel, enumModels := output.MarshalType(outputType)
		for _, model := range append(enumModels, baseModel) {
			var catalogType *client.CatalogTypeV2
			for _, existingCatalogType := range existingCatalogTypes {
				if model.TypeName == existingCatalogType.TypeName {
					catalogType = &existingCatalogType
					break
				}
			}
			if catalogType == nil {
				return fmt.Errorf("could not find catalog type for model '%s', this is a bug in the importer", model.TypeName)
			}

			catalogTypesByOutput[model.TypeName] = catalogType
		}
	}

	OUT("\n↻ Syncing catalog type schemas...")
	if opt.DryRun {
		for _, outputType := range cfg.Outputs() {
			baseModel, enumModels := output.MarshalType(outputType)
			for _, model := range append(enumModels, baseModel) {
				catalogType := catalogTypesByOutput[model.TypeName]

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
						var path *[]client.CatalogTypeAttributePathItemV2

						if attr.Path != nil {
							noPtrPath := *attr.Path
							newPath := lo.Map(noPtrPath, func(item client.CatalogTypeAttributePathItemPayloadV2, _ int) client.CatalogTypeAttributePathItemV2 {
								return client.CatalogTypeAttributePathItemV2{
									AttributeId: item.AttributeId,
								}
							})
							path = &newPath
						}

						updatedCatalogType.Categories = lo.Map(model.Categories, func(category string, _ int) client.CatalogTypeV2Categories {
							return client.CatalogTypeV2Categories(category)
						})

						updatedCatalogType.Schema.Attributes = append(updatedCatalogType.Schema.Attributes, client.CatalogTypeAttributeV2{
							Id:                *attr.Id,
							Name:              attr.Name,
							Type:              attr.Type,
							Array:             attr.Array,
							Mode:              client.CatalogTypeAttributeV2Mode(*attr.Mode),
							BacklinkAttribute: attr.BacklinkAttribute,
							Path:              path,
						})
					}
				}
				OUT("  ✔ %s (id=%s)", model.TypeName, catalogType.Id)

				// We only have attribute names in the response for a path attribute, not the
				// request. To avoid erroneous diffs, we strip the attribute names from any
				// path attributes.
				catalogTypeToCompare := *catalogType
				for _, attr := range catalogType.Schema.Attributes {
					if attr.Path != nil {
						for i := range *attr.Path {
							(*attr.Path)[i].AttributeName = ""
						}
					}
				}

				DIFF("  ", catalogTypeToCompare, updatedCatalogType)
			}
		}
	} else {
		// Update all the type schemas except for new derived attributes, which could reference
		// attributes that don't exist yet.
		catalogTypeVersions := map[string]int64{}
		for _, outputType := range cfg.Outputs() {
			baseModel, enumModels := output.MarshalType(outputType)
			for _, model := range append(enumModels, baseModel) {
				catalogType := catalogTypesByOutput[model.TypeName]

				attributesWithoutNewDerived := []client.CatalogTypeAttributePayloadV2{}
				for _, attr := range model.Attributes {
					isBacklink := *attr.Mode == client.CatalogTypeAttributePayloadV2ModeBacklink
					isPath := *attr.Mode == client.CatalogTypeAttributePayloadV2ModePath
					if isBacklink || isPath {
						_, inCurrentSchema := lo.Find(catalogType.Schema.Attributes, func(existingAttr client.CatalogTypeAttributeV2) bool {
							return existingAttr.Id == *attr.Id
						})
						if inCurrentSchema {
							attributesWithoutNewDerived = append(attributesWithoutNewDerived, attr)
						}
					} else {
						attributesWithoutNewDerived = append(attributesWithoutNewDerived, attr)
					}
				}

				categories := lo.Map(model.Categories, func(category string, _ int) client.UpdateTypeRequestBodyCategories {
					return client.UpdateTypeRequestBodyCategories(category)
				})

				logger.Log("msg", "updating catalog type", "catalog_type_id", catalogType.Id)
				result, err := cl.CatalogV2UpdateTypeWithResponse(ctx, catalogType.Id, client.CatalogV2UpdateTypeJSONRequestBody{
					Name:          model.Name,
					Description:   model.Description,
					Ranked:        &model.Ranked,
					Categories:    lo.ToPtr(categories),
					Annotations:   lo.ToPtr(getAnnotations(cfg.SyncID)),
					SourceRepoUrl: &opt.SourceRepoUrl,
				})
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("updating catalog type with name %s", model.TypeName))
				}

				version := result.JSON200.CatalogType.Schema.Version
				logger.Log("msg", "updating catalog type schema", "catalog_type_id", catalogType.Id, "version", version)
				schema, err := cl.CatalogV2UpdateTypeSchemaWithResponse(ctx, catalogType.Id, client.CatalogV2UpdateTypeSchemaJSONRequestBody{
					Version:    version,
					Attributes: attributesWithoutNewDerived,
				})
				if err != nil {
					return errors.Wrap(err, "updating catalog type schema")
				}

				catalogTypeVersions[catalogType.Id] = schema.JSON200.CatalogType.Schema.Version

				OUT("  ✔ %s (id=%s)", model.TypeName, catalogType.Id)
			}
		}

		// Then go through again and create any types that do have new derived attributes (backlinks or path)
		OUT("\n↻ Syncing derived attributes...")
		for _, outputType := range cfg.Outputs() {
			baseModel, enumModels := output.MarshalType(outputType)
			for _, model := range append(enumModels, baseModel) {
				catalogType := catalogTypesByOutput[model.TypeName]

				hasNewDerived := false
				for _, attr := range model.Attributes {
					if attr.Mode != nil && (attr.BacklinkAttribute != nil || attr.Path != nil) {
						_, inCurrentSchema := lo.Find(catalogType.Schema.Attributes, func(existingAttr client.CatalogTypeAttributeV2) bool {
							return existingAttr.Id == *attr.Id
						})

						if !inCurrentSchema {
							hasNewDerived = true
						}
					}
				}

				if !hasNewDerived {
					continue
				}
				version := catalogTypeVersions[catalogType.Id]
				logger.Log("msg", "updating catalog type schema: creating derived attribute(s)", "catalog_type_id", catalogType.Id, "version", version)

				_, err = cl.CatalogV2UpdateTypeSchemaWithResponse(ctx, catalogType.Id, client.CatalogV2UpdateTypeSchemaJSONRequestBody{
					Version:    version,
					Attributes: model.Attributes,
				})
				if err != nil {
					return errors.Wrap(err, "updating catalog type schema")
				}

				OUT("  ✔ %s (id=%s)", model.TypeName, catalogType.Id)
			}
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
				sourceLabel := lo.Must(source.Backend()).String()

				sourceEntries, err := source.Load(ctx, logger)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("loading entries from source: %s", sourceLabel))
				}

				for _, sourceEntry := range sourceEntries {
					parsedEntries, err := sourceEntry.Parse()
					if err != nil {
						sample := string(sourceEntry.Content)
						if len(sample) > opt.SampleLength {
							sample = sample[:opt.SampleLength]
						}
						logger.Log(
							"source", sourceEntry.Origin,
							"error", errors.Wrap(err, "parsing source entry"),
							"sample", sample,
						)
					}

					sourcedEntries = append(sourcedEntries, parsedEntries...)
				}

				OUT("    ✔ %s (found %d entries)", sourceLabel, len(sourcedEntries))
			}
		}

		OUT("\n  ↻ Syncing entries...")
		for idx, outputType := range pipeline.Outputs {
			OUT("\n    ↻ %s", outputType.TypeName)

			// Filter source for each of the output types
			entries, err := output.Collect(ctx, logger, outputType, sourcedEntries)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("outputs.%d (type_name='%s')", idx, outputType.TypeName))
			}
			OUT("      ✔ Building entries... (found %d entries matching filters)", len(entries))

			// Marshal entries using the JS expressions.
			entryModels, err := output.MarshalEntries(ctx, logger, outputType, entries)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("outputs.%d (type_name='%s')", idx, outputType.TypeName))
			}

			// As a precaution, error if we think there are no entries for this output and we
			// haven't explicitly permitted deleting all entries.
			if len(entryModels) == 0 && !opt.AllowDeleteAll {
				return errors.New(fmt.Sprintf("outputs (type_name = '%s'): found 0 matching entries and would delete everything but --allow-delete-all not set", outputType.TypeName))
			}

			// This can be reused for both model and enum types.
			entriesClient := newEntriesClient(cl, existingCatalogTypes, opt.DryRun)

			{
				logger.Log("msg", "reconciling catalog entries", "output", outputType.TypeName)
				catalogType := catalogTypesByOutput[outputType.TypeName]

				err = reconcile.Entries(ctx, logger, entriesClient, outputType, catalogType, entryModels, newEntriesProgress(!opt.DryRun), opt.CatalogEntriesAPIPageSize)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("outputs (type_name = '%s'): reconciling catalog entries", outputType.TypeName))
				}
			}

			// Process enum attributes, which require generating from the result of the parent
			// model's attribute.
			_, enumModels := output.MarshalType(outputType)
			for _, enumModel := range enumModels {
				// We've got an enum attribute, which means we need to sync the enum values.
				valueSet := map[string]bool{}
				for _, entry := range entryModels {
					value := entry.AttributeValues[enumModel.SourceAttribute.ID]
					if value.Value != nil {
						valueSet[*value.Value.Literal] = true
					}
					if value.ArrayValue != nil {
						for _, elementValue := range *value.ArrayValue {
							valueSet[*elementValue.Literal] = true
						}
					}
				}

				enumModels := []*output.CatalogEntryModel{}
				for value := range valueSet {
					enumModels = append(enumModels, &output.CatalogEntryModel{
						ExternalID:      value,
						Name:            value,
						Aliases:         []string{},
						AttributeValues: map[string]client.EngineParamBindingPayloadV2{},
					})
				}

				OUT("\n    ↻ %s (enum)", enumModel.TypeName)
				catalogType := catalogTypesByOutput[enumModel.TypeName]
				err := reconcile.Entries(ctx, logger, entriesClient, outputType, catalogType, enumModels, newEntriesProgress(!opt.DryRun), opt.CatalogEntriesAPIPageSize)
				if err != nil {
					return errors.Wrap(err,
						fmt.Sprintf("outputs (type_name = '%s'): enum for attribute (id = '%s'): %s: reconciling catalog entries",
							outputType.TypeName, enumModel.SourceAttribute.ID, enumModel.TypeName))
				}
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
		GetEntries: func(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV2, []client.CatalogEntryV2, error) {
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
			return reconcile.GetEntries(ctx, cl, catalogTypeID, pageSize)
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
				AttributeValues: map[string]client.EngineParamBindingPayloadV2{},
				ExternalId:      entry.ExternalId,
				Name:            entry.Name,
				Rank:            &entry.Rank,
			}
			if payload.Rank == nil && entry.Rank == 0 {
				existingPayload.Rank = nil
			}
			for attrID, attr := range entry.AttributeValues {
				result := client.EngineParamBindingPayloadV2{}
				if attr.Value != nil {
					result.Value = &client.EngineParamBindingValuePayloadV2{
						Literal: attr.Value.Literal,
					}
				}
				if attr.ArrayValue != nil {
					arrayValue := []client.EngineParamBindingValuePayloadV2{}
					for _, elementValue := range *attr.ArrayValue {
						arrayValue = append(arrayValue, client.EngineParamBindingValuePayloadV2{
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
