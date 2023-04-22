package config

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/incident-io/catalog-importer/output"
	"github.com/incident-io/catalog-importer/source"
)

type Config struct {
	SyncID    string      `json:"sync_id"`
	Pipelines []*Pipeline `json:"pipelines"`
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

func Validate(cfg *Config) error {
	return validation.ValidateStruct(cfg,
		validation.Field(&cfg.SyncID, validation.Required),
		validation.Field(&cfg.Pipelines),
	)
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
