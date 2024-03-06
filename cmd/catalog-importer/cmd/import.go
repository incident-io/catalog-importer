package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/config"
	"github.com/incident-io/catalog-importer/v2/output"
	"github.com/incident-io/catalog-importer/v2/source"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/guregu/null.v3"
)

type ImportOptions struct {
	APIEndpoint      string
	APIKey           string
	RunSync          bool
	RunSyncDryRun    bool
	Files            []string
	Name             string
	Description      string
	TypeName         string
	SourceExternalID string
	SourceName       string
}

func (opt *ImportOptions) Bind(cmd *kingpin.CmdClause) *ImportOptions {
	cmd.Flag("api-endpoint", "Endpoint of the incident.io API").
		Default("https://api.incident.io").
		Envar("INCIDENT_ENDPOINT").
		StringVar(&opt.APIEndpoint)
	cmd.Flag("api-key", "API key for incident.io").
		Envar("INCIDENT_API_KEY").
		StringVar(&opt.APIKey)
	cmd.Flag("run-sync", "Actually run the sync using the config produced by the import").
		BoolVar(&opt.RunSync)
	cmd.Flag("run-sync-dry-run", "If --run-sync, whether to do so in dry-run").
		BoolVar(&opt.RunSyncDryRun)
	cmd.Flag("local-file", "Which files to read content from, compatible with the local source").
		Required().
		StringsVar(&opt.Files)
	cmd.Flag("name", "What to name the resulting catalog type e.g Devices").
		Required().
		StringVar(&opt.Name)
	cmd.Flag("description", "What should be the description for the resulting catalog type").
		Required().
		StringVar(&opt.Description)
	cmd.Flag("type-name", `What to give as a type name for the resulting catalog type e.g. Custom["Devices"]`).
		Required().
		StringVar(&opt.TypeName)
	cmd.Flag("source-external-id", "What field of each source entry should be used as an external ID").
		Required().
		StringVar(&opt.SourceExternalID)
	cmd.Flag("source-name", "What field of each source entry should be used as an entry name").
		Required().
		StringVar(&opt.SourceName)

	return opt
}

func (opt *ImportOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	src := &source.SourceLocal{
		Files: opt.Files,
	}

	logger.Log("msg", "loading entries from files", "files", opt.Files)
	sourceEntries, err := src.Load(ctx, logger)
	if err != nil {
		return errors.Wrap(err, "reading source files")
	}

	entries := []source.Entry{}
	for _, sourceEntry := range sourceEntries {
		parsedEntries, err := sourceEntry.Parse()
		if err != nil {
			continue // quietly skip
		}

		entries = append(entries, parsedEntries...)
	}

	attributes := map[string]*output.Attribute{}
	for _, entry := range entries {
		for key, value := range entry {
			_, alreadyAdded := attributes[key]
			if alreadyAdded {
				continue
			}

			escapedKey, _ := json.Marshal(key)

			attributes[key] = &output.Attribute{
				ID:     key,
				Name:   key,
				Type:   null.StringFrom("String"),
				Array:  reflect.TypeOf(value).Kind() == reflect.Slice,
				Source: null.StringFrom(fmt.Sprintf("_[%s]", string(escapedKey))),
			}
		}
	}

	cfg := config.Config{
		SyncID: "one-time-import",
		Pipelines: []*config.Pipeline{
			{
				Sources: []*source.Source{
					{
						Local: src,
					},
				},
				Outputs: []*output.Output{
					{
						Name:        opt.Name,
						Description: opt.Description,
						TypeName:    opt.TypeName,
						Source: output.SourceConfig{
							Name:       opt.SourceName,
							ExternalID: opt.SourceExternalID,
						},
						Attributes: lo.Values(attributes),
					},
				},
			},
		},
	}

	BANNER("Pipeline that will import this file printed below")
	output, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshalling config")
	}

	fmt.Println(string(output))

	if opt.RunSync {
		syncOpt := *syncOptions
		syncOpt.APIEndpoint = opt.APIEndpoint
		syncOpt.APIKey = opt.APIKey
		syncOpt.AllowDeleteAll = true
		syncOpt.DryRun = opt.RunSyncDryRun

		if err := syncOpt.Run(ctx, logger, &cfg); err != nil {
			return errors.Wrap(err, "running sync")
		}
	}

	return nil
}
