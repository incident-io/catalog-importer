package output

import (
	"context"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/incident-io/catalog-importer/v2/expr"
	"github.com/incident-io/catalog-importer/v2/source"
	"github.com/pkg/errors"
)

// Collect filters the list of entries against the source filter on the output, returning
// a list of all entries which pass the filter. jsTimeout bounds each filter evaluation.
func Collect(ctx context.Context, logger kitlog.Logger, output *Output, entries []source.Entry, jsTimeout time.Duration) ([]source.Entry, error) {
	if !output.Source.Filter.Valid {
		return entries, nil // no-op, the filter is blank
	}

	src := output.Source.Filter.String

	filteredEntries := []source.Entry{}
	for _, entry := range entries {
		result, err := expr.EvaluateSingleValue[bool](ctx, logger, src, entry, jsTimeout)
		if err != nil {
			return nil, errors.Wrap(err, "evaluating filter for entry")
		}

		if result != nil && *result {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	return filteredEntries, nil
}
