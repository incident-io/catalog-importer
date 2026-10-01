package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/kit/log"
)

type ValidateOptions struct {
	ConfigFile string
}

func (opt *ValidateOptions) Bind(cmd *kingpin.CmdClause) *ValidateOptions {
	cmd.Flag("config", "Config file in either Jsonnet, YAML or JSON (e.g. importer.jsonnet)").
		StringVar(&opt.ConfigFile)

	return opt
}

func (opt *ValidateOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	cfg, err := loadConfigOrError(ctx, opt.ConfigFile)
	if err != nil {
		return err
	}

	invalid := 0
	for _, outputType := range cfg.Outputs() {
		for _, err := range outputType.CompileExpressions() {
			ALWAYS_OUT("✖ %s: %s", outputType.TypeName, err)
			invalid++
		}
	}
	if invalid > 0 {
		return errors.Errorf("found %d invalid Javascript expressions", invalid)
	}

	BANNER("Config printed below")
	output, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshalling config")
	}

	fmt.Println(string(output))

	return nil
}
