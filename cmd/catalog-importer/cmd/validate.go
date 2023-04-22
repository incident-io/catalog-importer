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

const ValidateUsage = `Validate importer configuration

	catalog-importer validate \
		--config config.yaml

`

type ValidateOptions struct {
}

func (opt *ValidateOptions) Bind(cmd *kingpin.CmdClause) *ValidateOptions {
	return opt
}

func (opt *ValidateOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	cfg, err := config.FileLoader(*configFile).Load(ctx)
	if err != nil {
		return errors.Wrap(err, "loading config")
	}
	if err := config.Validate(cfg); err != nil {
		banner("Config file is invalid!")

		// Print the validation error in JSON. Needs improving.
		data, _ := json.MarshalIndent(err, "", "  ")
		fmt.Println(string(data))
	}

	banner("Config printed below")
	output, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshalling config")
	}

	fmt.Println(string(output))

	return nil
}
