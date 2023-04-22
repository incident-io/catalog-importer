package output

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Output represents a catalog type that will be managed by this importer. It includes
// config for the resulting catalog type, along with catalog type attributes and how to
// build the values of said attributes from the sourced entries.
type Output struct {
	Name        string        `json:"name" doc:"The name of the resulting catalog type."`
	Description string        `json:"description" doc:"Human readable description of what this catalog type represents."`
	TypeName    string        `json:"type_name" doc:"The unique type name that represents this catalog type, usually the name in CamelCase with no spaces."`
	Source      *SourceConfig `json:"source"`
	Attributes  []*Attribute  `json:"attributes"`
}

func (o Output) Validate() error {
	return validation.ValidateStruct(&o,
		validation.Field(&o.Name, validation.Required),
		validation.Field(&o.Description, validation.Required),
		validation.Field(&o.TypeName, validation.Required, validation.Match(regexp.MustCompile(`^Custom\["[A-Z][a-zA-Z]*"\]$`))),
		validation.Field(&o.Source, validation.Required),
		validation.Field(&o.Attributes, validation.Required),
	)
}

// SourceConfig controls how we filter the source for this output's entries, and sets the
// external ID – used to uniquely identify an entry in the catalog – and the aliases of
// that entry from the source.
type SourceConfig struct {
	Filter     string   `json:"filter" doc:"Filter to apply to sourced entries in CEL."`
	Name       string   `json:"name" doc:"What field of the entry should be used as the entry name."`
	ExternalID string   `json:"external_id" doc:"What field of the entry should be used as the external ID."`
	Aliases    []string `json:"aliases" doc:"Optionally, what fields of the entry should be set as aliases."`
}

func (s SourceConfig) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Filter, validation.Required),
		validation.Field(&s.Name, validation.Required),
		validation.Field(&s.ExternalID, validation.Required),
	)
}

type Attribute struct {
	ID     string `json:"id" doc:"The stable identifier for this attribute, in kebab-case."`
	Name   string `json:"name" doc:"The name of this attribute as presented in the catalog."`
	Type   string `json:"type" doc:"Type of this catalog, set to the type name of custom or synced catalog types."`
	Array  bool   `json:"array" doc:"Set to true if this attribute should be an array (zero or more values)."`
	Source string `json:"source" doc:"If powered by a source, this is the expression that gives the attribute value."`
}

func (a Attribute) Validate() error {
	return validation.ValidateStruct(&a,
		validation.Field(&a.ID, validation.Required),
		validation.Field(&a.Name, validation.Required),
		validation.Field(&a.Type, validation.Required),
		validation.Field(&a.Source, validation.Required),
	)
}
