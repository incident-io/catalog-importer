package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	kitlog "github.com/go-kit/kit/log"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pkg/errors"
)

type SourceInline struct {
	Entries []map[string]any `json:"entries"`
}

func (s SourceInline) Validate() error {
	return validation.ValidateStruct(&s)
}

func (s SourceInline) String() string {
	return "inline" // we can't put all entries here, that would be too much
}

func (s SourceInline) Load(ctx context.Context, logger kitlog.Logger, _ *http.Client) ([]*SourceEntry, error) {
	entries := []*SourceEntry{}
	for idx, entry := range s.Entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return nil, errors.Wrap(err, "marshaling json")
		}

		entries = append(entries, &SourceEntry{
			Origin:   fmt.Sprintf("inline: entries.%d", idx),
			Filename: "",
			Content:  data,
		})
	}

	return entries, nil
}
