package source

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
)

type SourceLocal struct {
	Files []string `json:"files"`
}

func (s SourceLocal) Load(ctx context.Context) ([]*SourceEntry, error) {
	results := map[string]*SourceEntry{}
	for _, pattern := range s.Files {
		matches, err := filepath.Glob(pattern)
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
					Origin:  fmt.Sprintf("local: %s", match),
					Content: data,
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
