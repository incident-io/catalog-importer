package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/fatih/color"
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
	NoProgress                bool
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
	cmd.Flag("no-progress", "Disable progress bars (useful for cron jobs and output redirection)").
		BoolVar(&opt.NoProgress)

	return opt
}

func (opt *SyncOptions) Run(ctx context.Context, logger kitlog.Logger, cfg *config.Config) error {
	if opt.Prune && opt.DryRun {
		return errors.New("cannot use --dry-run with --prune")
	}
	if opt.Prune && len(opt.Targets) > 0 {
		return errors.New("cannot use --targets with --prune")
	}

	// If you're dry-running, and you have set --quiet, you're going to have a bad
	// time because the whole point of a dry run is to produce output!
	if *quiet && opt.DryRun {
		*quiet = false
		OUT("WARNING: --quiet has been ignored because --dry-run is set")
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
	result, err := cl.CatalogV3ListTypesWithResponse(ctx)
	if err != nil {
		return errors.Wrap(err, "listing catalog types")
	}
	OUT("✔ Connected to incident.io API (%s)", opt.APIEndpoint)

	existingCatalogTypes := []client.CatalogTypeV3{}
	unmanagedCatalogTypes := []client.CatalogTypeV3{}
	for _, catalogType := range result.JSON200.CatalogTypes {
		logger := kitlog.With(logger,
			"catalog_type_id", catalogType.Id,
			"catalog_type_name", catalogType.Name,
		)

		syncID, ok := catalogType.Annotations[AnnotationSyncID]
		if !ok {
			if catalogType.SourceRepoUrl == nil && catalogType.IsEditable {
				level.Debug(logger).Log("msg", "catalog type is editable and unmanaged",
					"catalog_type_id", catalogType.Id)
				unmanagedCatalogTypes = append(unmanagedCatalogTypes, catalogType)
			} else {
				level.Debug(logger).Log("msg", "ignoring catalog type as it managed elsewhere")
			}
		} else if syncID != cfg.SyncID {
			logger.Log("msg", "ignoring catalog type as it is managed by a different importer",
				"catalog_type_sync_id", syncID)
		} else {
			existingCatalogTypes = append(existingCatalogTypes, catalogType)
		}
	}

	logger.Log("msg", "found managed catalog types",
		"catalog_types", strings.Join(lo.Map(existingCatalogTypes, func(ct client.CatalogTypeV3, _ int) string {
			return ct.TypeName
		}), ", "))
	OUT("✔ Found %d catalog types, with %d that match our sync ID (%s)",
		len(result.JSON200.CatalogTypes), len(existingCatalogTypes), cfg.SyncID)

	// Remove unmanaged types
	if opt.Prune {
		OUT("\n↻ Prune enabled (--prune), removing types that are no longer in config...")

		toDestroy := []client.CatalogTypeV3{}
	nextCatalogType:
		for _, existingCatalogType := range existingCatalogTypes {
			logger := kitlog.With(logger,
				"type_name", existingCatalogType.TypeName,
				"catalog_type_id", existingCatalogType.Id,
			)

			for _, confType := range cfg.AllOutputTypes() {
				if confType.TypeName == existingCatalogType.TypeName {
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
				_, err := cl.CatalogV3DestroyTypeWithResponse(ctx, catalogType.Id)
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
	for _, model := range cfg.AllOutputTypes() {
		logger := kitlog.With(logger, "type_name", model.TypeName)

		for _, unmanagedCatalogType := range unmanagedCatalogTypes {
			if model.TypeName == unmanagedCatalogType.TypeName {
				logger.Log("msg", "found unmanaged catalog type: skipping create")
				continue createCatalogType
			}
		}

		for _, existingCatalogType := range existingCatalogTypes {
			if model.TypeName == existingCatalogType.TypeName {
				level.Debug(logger).Log("catalog type already exists")
				continue createCatalogType
			}
		}

		var createdCatalogType client.CatalogTypeV3
		if opt.DryRun {
			logger.Log("msg", "catalog type does not already exist, simulating create for --dry-run")
			createdCatalogType = client.CatalogTypeV3{
				Id:                  fmt.Sprintf("DRY-RUN-%s", model.TypeName),
				Name:                model.Name,
				Description:         model.Description,
				TypeName:            model.TypeName,
				UseNameAsIdentifier: model.UseNameAsIdentifier,
				SourceRepoUrl:       &opt.SourceRepoUrl,
			}
		} else {
			logger.Log("msg", "catalog type does not already exist, creating")
			categories := lo.Map(model.Categories, func(category string, _ int) client.CatalogCreateTypePayloadV3Categories {
				return client.CatalogCreateTypePayloadV3Categories(category)
			})

			result, err := cl.CatalogV3CreateTypeWithResponse(ctx, client.CatalogCreateTypePayloadV3{
				Name:                model.Name,
				Description:         model.Description,
				Ranked:              &model.Ranked,
				TypeName:            lo.ToPtr(model.TypeName),
				Categories:          lo.ToPtr(categories),
				Annotations:         lo.ToPtr(getAnnotations(cfg.SyncID)),
				UseNameAsIdentifier: lo.ToPtr(model.UseNameAsIdentifier),
				SourceRepoUrl:       &opt.SourceRepoUrl,
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

	// Prepare a lookup of catalog type by the output name for subsequent pipeline steps.
	catalogTypesByOutput := map[string]*client.CatalogTypeV3{}
	for _, model := range cfg.AllOutputTypes() {
		var catalogType *client.CatalogTypeV3
		for _, existingCatalogType := range existingCatalogTypes {
			if model.TypeName == existingCatalogType.TypeName {
				catalogType = &existingCatalogType
				break
			}
		}
		for _, unmanagedCatalogType := range unmanagedCatalogTypes {
			if model.TypeName == unmanagedCatalogType.TypeName {
				catalogType = &unmanagedCatalogType
				break
			}
		}

		if catalogType == nil {
			return fmt.Errorf("could not find catalog type for model '%s', this is a bug in the importer", model.TypeName)
		}

		catalogTypesByOutput[model.TypeName] = catalogType
	}

	OUT("\n↻ Syncing catalog type schemas...")
	if opt.DryRun {
		for _, model := range cfg.AllOutputTypes() {
			catalogType := catalogTypesByOutput[model.TypeName]

			var updatedCatalogType client.CatalogTypeV3
			logger.Log("msg", "dry-run active, which means we fake a response")
			updatedCatalogType = *catalogType // they start the same
			updatedCatalogType.UseNameAsIdentifier = model.UseNameAsIdentifier

			// Then we pretend like we've already updated the schema, which means we rebuild the
			// attributes.
			updatedCatalogType.Schema = client.CatalogTypeSchemaV3{
				Version:    updatedCatalogType.Schema.Version,
				Attributes: []client.CatalogTypeAttributeV3{},
			}
			for _, attr := range model.Attributes {
				var path *[]client.CatalogTypeAttributePathItemV3

				if attr.Path != nil {
					noPtrPath := *attr.Path
					newPath := lo.Map(noPtrPath, func(item client.CatalogTypeAttributePathItemPayloadV3, _ int) client.CatalogTypeAttributePathItemV3 {
						return client.CatalogTypeAttributePathItemV3{
							AttributeId: item.AttributeId,
						}
					})
					path = &newPath
				}

				updatedCatalogType.Categories = lo.Map(model.Categories, func(category string, _ int) client.CatalogTypeV3Categories {
					return client.CatalogTypeV3Categories(category)
				})

				updatedCatalogType.Schema.Attributes = append(updatedCatalogType.Schema.Attributes, client.CatalogTypeAttributeV3{
					Id:                *attr.Id,
					Name:              attr.Name,
					Type:              attr.Type,
					Array:             attr.Array,
					Mode:              client.CatalogTypeAttributeV3Mode(*attr.Mode),
					BacklinkAttribute: attr.BacklinkAttribute,
					Path:              path,
				})
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
	} else {
		// Update all the type schemas except for new derived attributes, which could reference
		// attributes that don't exist yet.
		catalogTypeVersions := map[string]int64{}
		for _, model := range cfg.AllOutputTypes() {
			catalogType := catalogTypesByOutput[model.TypeName]

			attributesWithoutNewDerived := []client.CatalogTypeAttributePayloadV3{}
			for _, attr := range model.Attributes {
				isBacklink := *attr.Mode == client.CatalogTypeAttributePayloadV3ModeBacklink
				isPath := *attr.Mode == client.CatalogTypeAttributePayloadV3ModePath
				if isBacklink || isPath {
					_, inCurrentSchema := lo.Find(catalogType.Schema.Attributes, func(existingAttr client.CatalogTypeAttributeV3) bool {
						return existingAttr.Id == *attr.Id
					})
					if inCurrentSchema {
						attributesWithoutNewDerived = append(attributesWithoutNewDerived, attr)
					}
				} else {
					attributesWithoutNewDerived = append(attributesWithoutNewDerived, attr)
				}
			}

			categories := lo.Map(model.Categories, func(category string, _ int) client.CatalogUpdateTypePayloadV3Categories {
				return client.CatalogUpdateTypePayloadV3Categories(category)
			})

			logger.Log("msg", "updating catalog type", "catalog_type_id", catalogType.Id)
			result, err := cl.CatalogV3UpdateTypeWithResponse(ctx, catalogType.Id, client.CatalogV3UpdateTypeJSONRequestBody{
				Name:                model.Name,
				Description:         model.Description,
				Ranked:              &model.Ranked,
				Categories:          lo.ToPtr(categories),
				Annotations:         lo.ToPtr(getAnnotations(cfg.SyncID)),
				UseNameAsIdentifier: lo.ToPtr(model.UseNameAsIdentifier),
				SourceRepoUrl:       &opt.SourceRepoUrl,
			})
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("updating catalog type with name %s", model.TypeName))
			}

			version := result.JSON200.CatalogType.Schema.Version
			logger.Log("msg", "updating catalog type schema", "catalog_type_id", catalogType.Id, "version", version)
			schemaJSON, _ := json.Marshal(attributesWithoutNewDerived)
			level.Debug(logger).Log("msg", "updating catalog type schema", "catalog_type_id", catalogType.Id, "schema", schemaJSON)
			schema, err := cl.CatalogV3UpdateTypeSchemaWithResponse(ctx, catalogType.Id, client.CatalogV3UpdateTypeSchemaJSONRequestBody{
				Version:    version,
				Attributes: attributesWithoutNewDerived,
			})
			if err != nil {
				return errors.Wrap(err, "updating catalog type schema")
			}

			catalogTypeVersions[catalogType.Id] = schema.JSON200.CatalogType.Schema.Version

			OUT("  ✔ %s (id=%s)", model.TypeName, catalogType.Id)
		}

		// Then go through again and create any types that do have new derived attributes (backlinks or path)
		OUT("\n↻ Syncing derived attributes...")
		for _, model := range cfg.AllOutputTypes() {
			catalogType := catalogTypesByOutput[model.TypeName]

			hasNewDerived := false
			for _, attr := range model.Attributes {
				if attr.Mode != nil && (attr.BacklinkAttribute != nil || attr.Path != nil) {
					_, inCurrentSchema := lo.Find(catalogType.Schema.Attributes, func(existingAttr client.CatalogTypeAttributeV3) bool {
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

			_, err = cl.CatalogV3UpdateTypeSchemaWithResponse(ctx, catalogType.Id, client.CatalogV3UpdateTypeSchemaJSONRequestBody{
				Version:    version,
				Attributes: model.Attributes,
			})
			if err != nil {
				return errors.Wrap(err, "updating catalog type schema")
			}

			OUT("  ✔ %s (id=%s)", model.TypeName, catalogType.Id)
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

				showProgress := !opt.DryRun && !opt.NoProgress
				err = reconcile.Entries(ctx, logger, entriesClient, outputType, catalogType, entryModels, newEntriesProgress(showProgress), opt.CatalogEntriesAPIPageSize)
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
						AttributeValues: map[string]client.CatalogEngineParamBindingPayloadV3{},
					})
				}

				OUT("\n    ↻ %s (enum)", enumModel.TypeName)
				catalogType := catalogTypesByOutput[enumModel.TypeName]
				showProgress := !opt.DryRun && !opt.NoProgress
				err := reconcile.Entries(ctx, logger, entriesClient, outputType, catalogType, enumModels, newEntriesProgress(showProgress), opt.CatalogEntriesAPIPageSize)
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
func newEntriesClient(cl *client.ClientWithResponses, existingCatalogTypes []client.CatalogTypeV3, dryRun bool) reconcile.EntriesClient {
	if !dryRun {
		return reconcile.EntriesClientFromClient(cl)
	}

	// Cache entries by ID for bulk update diff generation
	entriesByID := make(map[string]*client.CatalogEntryV3)

	return reconcile.EntriesClient{
		GetEntries: func(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV3, []client.CatalogEntryV3, error) {
			// We're in dry-run and this catalog type is yet to be created. We can't ask the API
			// for the entries of a type that doesn't exist, so we return the type we faked from
			// the dry-run create and an empty list of entries.
			if strings.HasPrefix(catalogTypeID, "DRY-RUN") {
				for _, existingType := range existingCatalogTypes {
					if existingType.Id == catalogTypeID {
						return &existingType, []client.CatalogEntryV3{}, nil
					}
				}

				return nil, nil, fmt.Errorf("could not find dry-run faked catalog type with id='%s'", catalogTypeID)
			}

			// We're just a normal catalog type, use the real client.
			catalogType, entries, err := reconcile.GetEntries(ctx, cl, catalogTypeID, pageSize)
			if err != nil {
				return nil, nil, err
			}

			// Cache entries for bulk update diff generation
			for i := range entries {
				entriesByID[entries[i].Id] = &entries[i]
			}

			return catalogType, entries, nil
		},
		Delete: func(ctx context.Context, entry *client.CatalogEntryV3) error {
			DIFF("      ", *entry, client.CatalogEntryV3{})
			return nil
		},
		Create: func(ctx context.Context, payload client.CatalogCreateEntryPayloadV3) (*client.CatalogEntryV3, error) {
			DIFF("      ", client.CatalogCreateEntryPayloadV3{}, payload)
			entry := &client.CatalogEntryV3{
				Id: fmt.Sprintf("DRY-RUN-%s", uuid.NewString()),
			}

			return entry, nil
		},
		BulkUpdate: func(ctx context.Context, catalogTypeID string, entries []client.PartialEntryPayloadV3, updateAttributes *[]string) error {
			for _, partialEntry := range entries {
				fmt.Println(color.New(color.FgYellow).Sprintf("    UPDATE: entry_id=%s", partialEntry.EntryId))

				// Find existing entry for comparison
				existingEntry, ok := entriesByID[partialEntry.EntryId]
				if !ok {
					fmt.Println(color.New(color.FgRed).Sprintf("      ERROR: could not find entry for diff"))
					continue
				}

				// Convert PartialEntryPayloadV3 to UpdatePayloadV3 for diff display
				payload := client.CatalogUpdateEntryPayloadV3{
					Name:             lo.FromPtrOr(partialEntry.Name, ""),
					Rank:             partialEntry.Rank,
					ExternalId:       partialEntry.ExternalId,
					Aliases:          partialEntry.Aliases,
					AttributeValues:  partialEntry.AttributeValues,
					UpdateAttributes: updateAttributes,
				}

				// Build existing payload for comparison
				existingPayload := client.CatalogUpdateEntryPayloadV3{
					Aliases:         lo.ToPtr(existingEntry.Aliases),
					AttributeValues: map[string]client.CatalogEngineParamBindingPayloadV3{},
					ExternalId:      existingEntry.ExternalId,
					Name:            existingEntry.Name,
					Rank:            &existingEntry.Rank,
				}
				if payload.Rank == nil && existingEntry.Rank == 0 {
					existingPayload.Rank = nil
				}
				for attrID, attr := range existingEntry.AttributeValues {
					result := client.CatalogEngineParamBindingPayloadV3{}
					if attr.Value != nil {
						result.Value = &client.CatalogEngineParamBindingValuePayloadV3{
							Literal: attr.Value.Literal,
						}
					}
					if attr.ArrayValue != nil {
						arrayValue := []client.CatalogEngineParamBindingValuePayloadV3{}
						for _, elementValue := range *attr.ArrayValue {
							arrayValue = append(arrayValue, client.CatalogEngineParamBindingValuePayloadV3{
								Literal: elementValue.Literal,
							})
						}
						result.ArrayValue = &arrayValue
					}
					existingPayload.AttributeValues[attrID] = result
				}

				DIFF("      ", existingPayload, payload)
			}
			return nil
		},
	}
}

// newEntriesProgress creates hooks to render progress into the terminal while reconciling
// catalog entries.
func newEntriesProgress(showBars bool) *reconcile.EntriesProgress {
	// `--quiet` suppresses all progress bars.
	if lo.FromPtr(quiet) {
		showBars = false
	}

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
