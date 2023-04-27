package source

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type SourceInline struct {
	Entries []map[string]any `json:"entries"`
}

func (s SourceInline) Load(ctx context.Context) ([]*SourceEntry, error) {
	entries := []*SourceEntry{}
	for idx, entry := range s.Entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return nil, errors.Wrap(err, "marshaling json")
		}

		entries = append(entries, &SourceEntry{
			Origin:  fmt.Sprintf("inline: entries.%d", idx),
			Content: data,
		})
	}

	return entries, nil
}
