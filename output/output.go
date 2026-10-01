package output

import (
	"fmt"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/incident-io/catalog-importer/v2/expr"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/guregu/null.v3"
)

// Output represents a catalog type that will be managed by this importer. It includes
// config for the resulting catalog type, along with catalog type attributes and how to
// build the values of said attributes from the sourced entries.
type Output struct {
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	TypeName            string       `json:"type_name"`
	Ranked              bool         `json:"ranked"`
	Color               null.String  `json:"color"`
	Icon                null.String  `json:"icon"`
	UseNameAsIdentifier bool         `json:"use_name_as_identifier"`
	Source              SourceConfig `json:"source"`
	Attributes          []*Attribute `json:"attributes"`
	Categories          []string     `json:"categories"`
	OwningTeamIDs       []string     `json:"owning_team_ids"`
}

func (o Output) Validate() error {
	return validation.ValidateStruct(&o,
		validation.Field(&o.Name, validation.Required),
		validation.Field(&o.Description, validation.Required),
		validation.Field(&o.TypeName, validation.Required, validation.Match(regexp.MustCompile(`^Custom\["[a-zA-Z0-9]+"\]$`))),
		validation.Field(&o.Source, validation.Required),
	)
}

// CompileExpressions checks every Javascript expression in the output is valid, without
// running any of them, and returns an error for each one that isn't.
func (o Output) CompileExpressions() []error {
	expressions := []lo.Tuple2[string, string]{
		lo.T2("source.name", o.Source.Name),
		lo.T2("source.external_id", o.Source.ExternalID),
	}
	if o.Source.Filter.Valid {
		expressions = append(expressions, lo.T2("source.filter", o.Source.Filter.String))
	}
	if o.Source.Rank.Valid {
		expressions = append(expressions, lo.T2("source.rank", o.Source.Rank.String))
	}
	for idx, alias := range o.Source.Aliases {
		expressions = append(expressions, lo.T2(fmt.Sprintf("source.aliases.%d", idx), alias))
	}
	for _, attr := range o.Attributes {
		expressions = append(expressions, lo.T2(fmt.Sprintf("attributes.%s", attr.ID), attr.SourceExpression()))
	}

	errs := []error{}
	for _, expression := range expressions {
		if err := expr.Compile(expression.B); err != nil {
			errs = append(errs, errors.Wrap(err, expression.A))
		}
	}

	return errs
}

// SourceConfig controls how we filter the source for this output's entries, and sets the
// external ID – used to uniquely identify an entry in the catalog – and the aliases of
// that entry from the source.
type SourceConfig struct {
	Filter     null.String `json:"filter"`
	Name       string      `json:"name"`
	ExternalID string      `json:"external_id"`
	Rank       null.String `json:"rank"`
	Aliases    []string    `json:"aliases"`
}

func (s SourceConfig) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Name, validation.Required),
		validation.Field(&s.ExternalID, validation.Required),
	)
}

type Attribute struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Type              null.String    `json:"type"`
	Array             bool           `json:"array"`
	Source            null.String    `json:"source"`
	Enum              *AttributeEnum `json:"enum"`
	BacklinkAttribute null.String    `json:"backlink_attribute"`
	Path              []string       `json:"path"`
	SchemaOnly        bool           `json:"schema_only"`
}

func (a Attribute) Validate() error {
	return validation.ValidateStruct(&a,
		validation.Field(&a.ID, validation.Required),
		validation.Field(&a.Name, validation.Required),
		validation.Field(&a.Type,
			validation.Required.When(a.Enum == nil).Error("type is required when enum is not set"),
			validation.Empty.When(a.Enum != nil).Error("type cannot be set when enum is provided"),
		),
		validation.Field(&a.Enum,
			validation.Required.When(!a.Type.Valid).Error("enum is required if type is not set"),
			validation.Empty.When(a.Type.Valid).Error("enum cannot be provided when type is set"),
		),
	)
}

// SourceExpression is the expression that builds this attribute's value, defaulting to
// the field with the same ID as the attribute.
func (a Attribute) SourceExpression() string {
	if a.Source.Valid {
		return a.Source.String
	}

	return "$." + a.ID
}

func (a Attribute) IncludeInPayload() bool {
	if a.SchemaOnly {
		// These are left for the dashboard to set
		return false
	}
	if a.BacklinkAttribute.Valid {
		// Automatically set by the backlink
		return false
	}
	if a.Path != nil {
		// These are derived from other attributes
		return false
	}

	return true
}

type AttributeEnum struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	TypeName       string `json:"type_name"`
	EnableBacklink bool   `json:"enable_backlink"`
}
