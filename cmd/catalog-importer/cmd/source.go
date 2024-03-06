package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/output"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/yaml.v2"
)

type SourceOptions struct {
	ConfigFile   string
	SampleLength int
}

func (opt *SourceOptions) Bind(cmd *kingpin.CmdClause) *SourceOptions {
	cmd.Flag("config", "Config file in either Jsonnet, YAML or JSON (e.g. importer.jsonnet)").
		StringVar(&opt.ConfigFile)
	cmd.Flag("sample-length", "How many character to sample when logging about invalid source entries (for --debug only)").
		Default("256").
		IntVar(&opt.SampleLength)

	return opt
}

func (opt *SourceOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	// Load config
	cfg, err := loadConfigOrError(ctx, opt.ConfigFile)
	if err != nil {
		return err
	}

	for _, pipeline := range cfg.Pipelines {
		OUT("\n↻ Processing pipeline... (%s)", strings.Join(lo.Map(pipeline.Outputs, func(op *output.Output, _ int) string {
			return op.TypeName
		}), ", "))

		// Load entries from source
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

					for _, entry := range parsedEntries {
						data, err := yaml.Marshal(entry)
						if err != nil {
							return errors.Wrap(err, "marshaling YAML")
						}

						OUT("---\n" + string(data))
					}
				}
			}
		}
	}

	return nil
}
