package config

import (
	"context"
	"fmt"

	_ "embed"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/incident-io/catalog-importer/v2/output"
	"github.com/incident-io/catalog-importer/v2/source"
	"github.com/samber/lo"
)

//go:embed reference.jsonnet
var ReferenceConfig []byte

// We check the reference config file validates whenever we boot the binary. This is an
// aggressive check to ensure the reference is up-to-date, and is probably nicer as a test
// in future.
func init() {
	_, err := Parse("reference.jsonnet", ReferenceConfig)
	if err != nil {
		panic(fmt.Sprintf("failed to parse config/reference.jsonnet: %v", err))
	}
}

type Config struct {
	SyncID    string      `json:"sync_id,omitempty"`
	Pipelines []*Pipeline `json:"pipelines"`
}

func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.SyncID, validation.Required.
			Error("must provide a sync_id to track which resources are managed by this config, and to support clean-up when an output is removed")),
		validation.Field(&c.Pipelines),
	)
}

// Filter return a new config adjusted so all that remains is configuration pertaining to
// the given type names.
func (c Config) Filter(typeNames []string) *Config {
	clone := c
	for idx := range clone.Pipelines {
		clone.Pipelines[idx].Outputs = lo.Filter(clone.Pipelines[idx].Outputs, func(output *output.Output, _ int) bool {
			for _, target := range typeNames {
				if target == output.TypeName {
					return true
				}
			}

			return false
		})
	}

	clone.Pipelines = lo.Filter(clone.Pipelines, func(pipeline *Pipeline, _ int) bool {
		return len(pipeline.Outputs) > 0
	})

	return &clone
}

func (c Config) Outputs() []*output.Output {
	outputs := []*output.Output{}
	for _, pipeline := range c.Pipelines {
		outputs = append(outputs, pipeline.Outputs...)
	}

	return outputs
}

// Load returns currently loaded config
func (c Config) Load(context.Context) (Config, error) {
	return Config(c), nil
}

type Pipeline struct {
	Sources []*source.Source `json:"sources"`
	Outputs []*output.Output `json:"outputs"`
}

func (p Pipeline) Validate() error {
	return validation.ValidateStruct(&p)
}
