package output

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"gopkg.in/guregu/null.v3"
)

// Output represents a catalog type that will be managed by this importer. It includes
// config for the resulting catalog type, along with catalog type attributes and how to
// build the values of said attributes from the sourced entries.
type Output struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	TypeName    string       `json:"type_name"`
	Source      SourceConfig `json:"source"`
	Attributes  []*Attribute `json:"attributes"`
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
	Filter     null.String `json:"filter"`
	Name       string      `json:"name"`
	ExternalID string      `json:"external_id"`
	Aliases    []string    `json:"aliases"`
}

func (s SourceConfig) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Name, validation.Required),
		validation.Field(&s.ExternalID, validation.Required),
	)
}

type Attribute struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	Array  bool        `json:"array"`
	Source null.String `json:"source"`
}

func (a Attribute) Validate() error {
	return validation.ValidateStruct(&a,
		validation.Field(&a.ID, validation.Required),
		validation.Field(&a.Name, validation.Required),
		validation.Field(&a.Type, validation.Required),
	)
}
