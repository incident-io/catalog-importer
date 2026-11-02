package config

import (
	"context"
	"errors"
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
		validation.Field(&c.Pipelines, validation.Required, validation.Length(1, 0).
			Error("must specify at least one pipeline")),
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
		clone.Pipelines[idx].SyncedTypeOutputs = lo.Filter(clone.Pipelines[idx].SyncedTypeOutputs, func(output *output.SyncedTypeOutput, _ int) bool {
			return lo.Contains(typeNames, output.TypeName)
		})
	}

	clone.Pipelines = lo.Filter(clone.Pipelines, func(pipeline *Pipeline, _ int) bool {
		return len(pipeline.Outputs) > 0 || len(pipeline.SyncedTypeOutputs) > 0
	})

	return &clone
}

func (c Config) AllOutputTypes() []*output.CatalogTypeModel {
	types := []*output.CatalogTypeModel{}
	for _, outputType := range c.Outputs() {
		baseType, enumTypes := output.MarshalType(outputType)
		types = append(types, baseType)
		types = append(types, enumTypes...)
	}
	return types
}

func (c Config) Outputs() []*output.Output {
	outputs := []*output.Output{}
	for _, pipeline := range c.Pipelines {
		outputs = append(outputs, pipeline.Outputs...)
	}

	return outputs
}

func (c Config) SyncedTypeOutputs() []*output.SyncedTypeOutput {
	outputs := []*output.SyncedTypeOutput{}
	for _, pipeline := range c.Pipelines {
		outputs = append(outputs, pipeline.SyncedTypeOutputs...)
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
	// SyncedTypeOutputs write custom aliases and attributes onto existing entries of
	// catalog types that incident.io syncs, rather than creating a type of their own.
	SyncedTypeOutputs []*output.SyncedTypeOutput `json:"synced_type_outputs,omitempty"`
}

func (p Pipeline) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Sources, validation.Required, validation.Length(1, 0).
			Error("must specify at least one source")),
		validation.Field(&p.Outputs, validation.By(func(any) error {
			if len(p.Outputs) == 0 && len(p.SyncedTypeOutputs) == 0 {
				return errors.New("must specify at least one output or synced type output")
			}
			return nil
		})),
		validation.Field(&p.SyncedTypeOutputs),
	)
}

// TypeNames lists the catalog types this pipeline writes to.
func (p Pipeline) TypeNames() []string {
	typeNames := lo.Map(p.Outputs, func(op *output.Output, _ int) string {
		return op.TypeName
	})
	for _, op := range p.SyncedTypeOutputs {
		typeNames = append(typeNames, op.TypeName)
	}

	return typeNames
}
