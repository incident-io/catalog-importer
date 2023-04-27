package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/incident-io/catalog-importer/config"
	"github.com/pkg/errors"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/kit/log"
)

type ValidateOptions struct {
	ConfigFile string
}

func (opt *ValidateOptions) Bind(cmd *kingpin.CmdClause) *ValidateOptions {
	cmd.Flag("config", "Config file in either Jsonnet, YAML or JSON (e.g. config.jsonnet)").
		Required().
		StringVar(&opt.ConfigFile)

	return opt
}

func (opt *ValidateOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	cfg, err := config.FileLoader(opt.ConfigFile).Load(ctx)
	if err != nil {
		return errors.Wrap(err, "loading config")
	}
	if err := cfg.Validate(); err != nil {
		BANNER("Config file is invalid!")

		// Print the validation error in JSON. Needs improving.
		data, _ := json.MarshalIndent(err, "", "  ")
		fmt.Println(string(data))
	}

	BANNER("Config printed below")
	output, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshalling config")
	}

	fmt.Println(string(output))

	return nil
}
