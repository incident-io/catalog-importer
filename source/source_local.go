package source

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/pkg/errors"
	"github.com/yargevad/filepathx"
)

type SourceLocal struct {
	Files []string `json:"files"`
}

func (s SourceLocal) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Files, validation.Length(1, 0).
			Error("must provide at least one file when using local source")),
	)
}

func (s SourceLocal) String() string {
	return fmt.Sprintf("local (files=%s)", strings.Join(s.Files, ","))
}

func (s SourceLocal) Load(ctx context.Context, logger kitlog.Logger, _ *http.Client) ([]*SourceEntry, error) {
	results := map[string]*SourceEntry{}
	for _, pattern := range s.Files {
		matches, err := filepathx.Glob(pattern)
		if err != nil {
			return nil, errors.Wrap(err, "glob matching files")
		}

		if len(matches) == 0 {
			return nil, errors.Errorf("no files found matching pattern: %s", pattern)
		}

		for _, match := range matches {
			_, ok := results[match]
			if !ok {
				data, err := os.ReadFile(match)
				if err != nil {
					return nil, errors.Wrap(err, fmt.Sprintf("reading file: %s", match))
				}

				results[match] = &SourceEntry{
					Origin:   fmt.Sprintf("local: %s", match),
					Filename: match,
					Content:  data,
				}
			}
		}
	}

	entries := []*SourceEntry{}
	for _, result := range results {
		entries = append(entries, result)
	}

	return entries, nil
}
