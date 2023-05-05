package source

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

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

func (s SourceLocal) Load(ctx context.Context) ([]*SourceEntry, error) {
	results := map[string]*SourceEntry{}
	for _, pattern := range s.Files {
		matches, err := filepathx.Glob(pattern)
		if err != nil {
			return nil, errors.Wrap(err, "glob matching files")
		}

		for _, match := range matches {
			_, ok := results[match]
			if !ok {
				data, err := ioutil.ReadFile(match)
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
