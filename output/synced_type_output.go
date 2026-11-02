package output

import (
	"fmt"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/expr"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/guregu/null.v3"
)

// SyncedTypeOutput writes custom aliases and extra attributes onto the entries of a
// catalog type that incident.io syncs from an integration, such as Schedule,
// EscalationPath or User.
//
// The integration owns those entries: it creates and deletes them, and sets their name,
// external ID, aliases and synced attributes. This output never does any of that. It
// matches each source entry to an existing entry, then writes only the custom aliases and
// the attributes it declares.
type SyncedTypeOutput struct {
	TypeName   string                 `json:"type_name"`
	Source     SyncedTypeSourceConfig `json:"source"`
	Attributes []*SyncedTypeAttribute `json:"attributes"`
}

func (o SyncedTypeOutput) Validate() error {
	return validation.ValidateStruct(&o,
		validation.Field(&o.TypeName, validation.Required,
			validation.By(func(any) error {
				if strings.HasPrefix(o.TypeName, "Custom[") {
					return errors.New("must be a catalog type that incident.io syncs, such as Schedule: use outputs for Custom types")
				}
				return nil
			}),
		),
		validation.Field(&o.Source, validation.Required),
		validation.Field(&o.Attributes, validation.By(func(any) error {
			if o.Source.CustomAliases == nil && len(o.Attributes) == 0 {
				return errors.New("must set source.custom_aliases or at least one attribute, or this output has nothing to write")
			}

			seen := map[string]bool{}
			for _, attr := range o.Attributes {
				if attr != nil && seen[attr.ID] {
					return fmt.Errorf("attribute %q is listed more than once", attr.ID)
				}
				if attr != nil {
					seen[attr.ID] = true
				}
			}

			return nil
		})),
	)
}

// CompileExpressions checks every Javascript expression in the output is valid, without
// running any of them, and returns an error for each one that isn't.
func (o SyncedTypeOutput) CompileExpressions() []error {
	expressions := []lo.Tuple2[string, string]{
		lo.T2("source.match", o.Source.Match),
	}
	if o.Source.Filter.Valid {
		expressions = append(expressions, lo.T2("source.filter", o.Source.Filter.String))
	}
	for idx, alias := range o.Source.CustomAliases {
		expressions = append(expressions, lo.T2(fmt.Sprintf("source.custom_aliases.%d", idx), alias))
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

// SyncedTypeSourceConfig controls which source entries this output uses, and how each one
// finds the existing entry it writes to.
type SyncedTypeSourceConfig struct {
	Filter null.String `json:"filter"`
	// Match evaluates to the ID, external ID, name or alias of an existing entry.
	Match string `json:"match"`
	// CustomAliases builds the full list of custom aliases for the matched entry. When
	// nil, this output leaves custom aliases alone. When set, even to an empty list, this
	// output owns the whole list and removes any custom alias it doesn't produce.
	CustomAliases []string `json:"custom_aliases"`
}

func (s SyncedTypeSourceConfig) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Match, validation.Required),
	)
}

// SyncedTypeAttribute is an attribute that already exists on the synced catalog type,
// which this output sets the value of. Its type comes from the catalog type's schema.
type SyncedTypeAttribute struct {
	ID     string      `json:"id"`
	Source null.String `json:"source"`
}

func (a SyncedTypeAttribute) Validate() error {
	return validation.ValidateStruct(&a,
		validation.Field(&a.ID, validation.Required),
	)
}

// SourceExpression is the expression that builds this attribute's value, defaulting to
// the field with the same ID as the attribute.
func (a SyncedTypeAttribute) SourceExpression() string {
	if a.Source.Valid {
		return a.Source.String
	}

	return "$." + a.ID
}

// ResolveAttributes checks the declared attributes against the schema of the synced
// catalog type, and returns them with the type and array settings from that schema so
// their values can be built.
//
// An API key can only write attributes in api or dashboard mode. The others are set by
// incident.io, and the API drops any value sent for them without an error, so we reject
// them here where we can say why.
func (o SyncedTypeOutput) ResolveAttributes(catalogType *client.CatalogTypeV3) ([]*Attribute, error) {
	writable := lo.Filter(catalogType.Schema.Attributes, func(attr client.CatalogTypeAttributeV3, _ int) bool {
		return attr.Mode == client.CatalogTypeAttributeV3ModeApi || attr.Mode == client.CatalogTypeAttributeV3ModeDashboard
	})
	describeWritable := func() string {
		if len(writable) == 0 {
			return "this type has no attributes you can set. Add one in the incident.io dashboard first"
		}

		return "attributes you can set are: " + strings.Join(lo.Map(writable, func(attr client.CatalogTypeAttributeV3, _ int) string {
			return fmt.Sprintf("%s (%s)", attr.Id, attr.Name)
		}), ", ")
	}

	attributes := []*Attribute{}
	for _, declared := range o.Attributes {
		schemaAttr, found := lo.Find(catalogType.Schema.Attributes, func(attr client.CatalogTypeAttributeV3) bool {
			return attr.Id == declared.ID
		})
		if !found {
			return nil, fmt.Errorf("attribute %q does not exist on %s: %s", declared.ID, catalogType.TypeName, describeWritable())
		}

		if !lo.ContainsBy(writable, func(attr client.CatalogTypeAttributeV3) bool { return attr.Id == declared.ID }) {
			return nil, fmt.Errorf("attribute %q (%s) on %s is set by incident.io (mode %s), so the importer can't write it: %s",
				declared.ID, schemaAttr.Name, catalogType.TypeName, schemaAttr.Mode, describeWritable())
		}

		attributes = append(attributes, &Attribute{
			ID:     schemaAttr.Id,
			Name:   schemaAttr.Name,
			Type:   null.StringFrom(schemaAttr.Type),
			Array:  schemaAttr.Array,
			Source: null.StringFrom(declared.SourceExpression()),
		})
	}

	return attributes, nil
}
