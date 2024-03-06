package output

import (
	"context"

	"github.com/incident-io/catalog-importer/v2/expr"
	"github.com/incident-io/catalog-importer/v2/source"
	"github.com/pkg/errors"
)

// Collect filters the list of entries against the source filter on the output, returning
// a list of all entries which pass the filter.
func Collect(ctx context.Context, output *Output, entries []source.Entry) ([]source.Entry, error) {
	if !output.Source.Filter.Valid {
		return entries, nil // no-op, the filter is blank
	}

	src := output.Source.Filter.String

	filteredEntries := []source.Entry{}
	for _, entry := range entries {
		result, err := expr.EvaluateSingleValue[bool](ctx, src, entry)
		if err != nil {
			return nil, errors.Wrap(err, "evaluating filter for entry")
		}

		if result {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	return filteredEntries, nil
}
