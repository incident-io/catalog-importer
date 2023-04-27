package config

import (
	"context"
	"fmt"

	_ "embed"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/incident-io/catalog-importer/output"
	"github.com/incident-io/catalog-importer/source"
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
	SyncID    string      `json:"sync_id"`
	Pipelines []*Pipeline `json:"pipelines"`
}

func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.SyncID, validation.Required),
		validation.Field(&c.Pipelines),
	)
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
	return validation.ValidateStruct(&p,
		validation.Field(&p.Sources, validation.Required),
		validation.Field(&p.Outputs, validation.Required),
	)
}
