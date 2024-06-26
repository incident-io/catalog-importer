package output

import (
	"context"
	"fmt"

	kitlog "github.com/go-kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/expr"
	"github.com/incident-io/catalog-importer/v2/source"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type CatalogTypeModel struct {
	Name            string
	Description     string
	TypeName        string
	Ranked          bool
	Attributes      []client.CatalogTypeAttributePayloadV2
	Categories      []string
	SourceAttribute *Attribute // tracks the origin attribute, if an enum model
	SourceRepoUrl   string
}

type CatalogEntryModel struct {
	ExternalID      string
	Name            string
	Aliases         []string
	Rank            int32
	AttributeValues map[string]client.EngineParamBindingPayloadV2
}

// MarshalType builds the base catalog type model for the output, and any associated enum
// types for its attributes.
func MarshalType(output *Output) (base *CatalogTypeModel, enumTypes []*CatalogTypeModel) {
	base = &CatalogTypeModel{
		Name:        output.Name,
		Description: output.Description,
		TypeName:    output.TypeName,
		Ranked:      output.Ranked,
		Attributes:  []client.CatalogTypeAttributePayloadV2{},
		Categories:  output.Categories,
	}
	for _, attr := range output.Attributes {
		var attrType string

		// When an attribute is an enum you don't need to define the type as we'll load it
		// from the enum definition.
		if attr.Enum != nil {
			attrType = attr.Enum.TypeName
		} else {
			attrType = attr.Type.String
		}

		var mode client.CatalogTypeAttributePayloadV2Mode
		switch {
		case attr.BacklinkAttribute.Valid:
			mode = client.CatalogTypeAttributePayloadV2ModeBacklink
		case attr.Path != nil:
			mode = client.CatalogTypeAttributePayloadV2ModePath
		default:
			mode = client.CatalogTypeAttributePayloadV2ModeManual
		}

		attribute := client.CatalogTypeAttributePayloadV2{
			Id:                lo.ToPtr(attr.ID),
			Name:              attr.Name,
			Type:              attrType,
			Array:             attr.Array,
			BacklinkAttribute: attr.BacklinkAttribute.Ptr(),
			Mode:              lo.ToPtr(mode),
		}

		if attr.Path != nil {
			attribute.Path = lo.ToPtr(lo.Map(attr.Path, func(item string, _ int) client.CatalogTypeAttributePathItemPayloadV2 {
				return client.CatalogTypeAttributePathItemPayloadV2{
					AttributeId: item,
				}
			}))
		}

		base.Attributes = append(base.Attributes, attribute)

		// The enums we generate should be returned as types too, as we'll need to sync them
		// just as any other.
		if attr.Enum != nil {
			enumTypes = append(enumTypes, &CatalogTypeModel{
				Name:        attr.Enum.Name,
				Description: attr.Enum.Description,
				TypeName:    attr.Enum.TypeName,
				Ranked:      output.Ranked,
				Attributes: []client.CatalogTypeAttributePayloadV2{
					{
						Id:   lo.ToPtr("description"),
						Name: "Description",
						Type: "String",
						Mode: lo.ToPtr(client.CatalogTypeAttributePayloadV2ModeManual),
					},
				},
				SourceAttribute: lo.ToPtr(*attr),
			})
		}
	}

	return
}

// MarshalEntries builds payloads to for the entries of the given output, assuming those
// entries have already been filtered.
//
// The majority of the work comes from compiling and evaluating the JS expressions that
// marshal the catalog entries from source.
func MarshalEntries(ctx context.Context, logger kitlog.Logger, output *Output, entries []source.Entry) ([]*CatalogEntryModel, error) {
	nameSource := output.Source.Name
	externalIDSource := output.Source.ExternalID
	aliasesSource := output.Source.Aliases

	var (
		attributeByID    = map[string]*Attribute{}
		attributeSources = map[string]string{}
	)
	for _, attr := range output.Attributes {
		// Use the attribute ID by default if source isn't explicitly provided.
		source := "$." + attr.ID
		if attr.Source.Valid {
			source = attr.Source.String
		}
		attributeByID[attr.ID] = attr
		attributeSources[attr.ID] = source
	}

	catalogEntryModels := []*CatalogEntryModel{}
	for _, entry := range entries {
		name, err := expr.EvaluateSingleValue[string](ctx, logger, nameSource, entry)
		if err != nil {
			return nil, errors.Wrap(err, "evaluating entry name")
		}

		externalID, err := expr.EvaluateSingleValue[string](ctx, logger, externalIDSource, entry)
		if err != nil {
			return nil, errors.Wrap(err, "evaluating entry external ID")
		}

		var rank *int
		if rankSource := output.Source.Rank; rankSource.Valid && rankSource.String != "" {
			var err error
			rank, err = expr.EvaluateSingleValue[int](ctx, logger, rankSource.String, entry)
			if err != nil {
				return nil, errors.Wrap(err, "evaluating entry rank")
			}
		}

		// Try to parse each alias as either a string or a string array, then concat and
		// dedupe them together.
		aliases := []string{}
		for idx, aliasSource := range aliasesSource {
			toAdd := []string{}
			alias, err := expr.EvaluateSingleValue[string](ctx, logger, aliasSource, entry)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("aliases.%d: evaluating entry alias", idx))
			}
			if alias == nil {
				aliasArray, arrayErr := expr.EvaluateArray[string](ctx, logger, aliasSource, entry)
				if arrayErr != nil {
					return nil, errors.Wrap(err, fmt.Sprintf("aliases.%d: evaluating entry alias", idx))
				}
				toAdd = append(toAdd, aliasArray...)
			} else {
				toAdd = append(toAdd, *alias)
			}

			for _, alias := range toAdd {
				if alias != "" {
					aliases = append(aliases, alias)
				}
			}
		}

		// Attribute values are built best effort, as it might not be the case that upstream
		// source entries have these fields, or have fields of the correct type.
		attributeValues := map[string]client.EngineParamBindingPayloadV2{}

		for attributeID, src := range attributeSources {
			binding := client.EngineParamBindingPayloadV2{}

			if attributeByID[attributeID].Array {
				valueLiterals, err := expr.EvaluateArray[any](ctx, logger, src, entry)
				if err != nil {
					return catalogEntryModels, errors.Wrap(err, "evaluating attribute")
				}
				if valueLiterals == nil {
					continue
				}

				arrayValue := []client.EngineParamBindingValuePayloadV2{}
				for _, literalAny := range valueLiterals {
					literal, ok := literalAny.(string)
					if !ok {
						continue
					}

					arrayValue = append(arrayValue, client.EngineParamBindingValuePayloadV2{
						Literal: lo.ToPtr(literal),
					})
				}

				binding.ArrayValue = &arrayValue
			} else {
				literal, err := evaluateEntryWithAttributeType(ctx, src, entry, attributeByID[attributeID], logger)
				if err != nil {
					return catalogEntryModels, errors.Wrap(err, "evaluating attribute")
				}
				if literal == nil {
					continue
				}

				binding.Value = &client.EngineParamBindingValuePayloadV2{
					Literal: literal,
				}
			}

			attributeValues[attributeID] = binding
		}

		catalogEntryModel := CatalogEntryModel{
			Aliases:         aliases,
			AttributeValues: attributeValues,
		}
		if name != nil {
			catalogEntryModel.Name = *name
		}
		if externalID != nil {
			catalogEntryModel.ExternalID = *externalID
		}
		if rank != nil {
			catalogEntryModel.Rank = int32(*rank)
		}
		catalogEntryModels = append(catalogEntryModels, &catalogEntryModel)
	}

	return catalogEntryModels, nil
}

func evaluateEntryWithAttributeType(ctx context.Context, src string, entry map[string]any, attribute *Attribute, logger kitlog.Logger) (*string, error) {
	var literal *string

	// If we have an attribute type of type Bool or Number, we can try to evaluate the program against the scope
	// with the appropriate type.
	// If that fails, we'll fall back to a string literal since we accept passing a boolean or numeric value
	// as a string literal.
	if attribute != nil && attribute.Type.Valid {
		switch attribute.Type.String {
		case "Bool":
			literal, _ = evaluateEntryWithType[bool](ctx, src, entry, logger)
			if literal != nil {
				return literal, nil
			}
		case "Number":
			// Number accepts float or int, so we'll try to evaluate as a float first.
			literal, _ = evaluateEntryWithType[float64](ctx, src, entry, logger)
			if literal != nil {
				return literal, nil
			}
			literal, _ = evaluateEntryWithType[int64](ctx, src, entry, logger)
			if literal != nil {
				return literal, nil
			}
		}
	}

	// If we have an attribute type of type String, or we failed to evaluate the program against the scope
	// with the appropriate type, we'll try to evaluate as a string literal.
	return evaluateEntryWithType[string](ctx, src, entry, logger)
}

func evaluateEntryWithType[ReturnType any](ctx context.Context, src string, entry map[string]any, logger kitlog.Logger) (*string, error) {
	literal, err := expr.EvaluateSingleValue[ReturnType](ctx, logger, src, entry)
	if err != nil {
		return nil, err
	}
	if literal == nil {
		return nil, nil
	}

	return lo.ToPtr(fmt.Sprintf("%v", *literal)), nil
}
