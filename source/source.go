package source

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// SourceEntry is an entry that has been discovered in a source, with the contents of the
// source file and an Origin that explains where the entry came from, specific to the type
// of source that produced it.
type SourceEntry struct {
	Origin  string
	Content []byte
}

// Entry is a single sourced entry.  It's just a basic map, but makes it much clearer when
// building lists of this type, as the type syntax can get a bit messy.
type Entry map[string]any

func (e SourceEntry) Parse() ([]Entry, error) {
	var (
		entries   = []Entry{}
		docChunks = bytes.Split(e.Content, []byte("\n---"))
	)
	for _, chunk := range docChunks {
		entry := map[string]any{}
		if err := yaml.Unmarshal(chunk, &entry); err != nil {
			return nil, errors.Wrap(err, "parsing YAML")
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// Source is instantiated from configuration and represents a source of catalog files.
type Source struct {
	Local  *SourceLocal  `json:"local,omitempty"`
	Inline *SourceInline `json:"inline,omitempty" doc:"Define entries on this source to load them directly."`
}

func (s Source) Name() string {
	if s.Local != nil {
		return "local"
	}
	if s.Inline != nil {
		return "inline"
	}

	return "unknown"
}

var ErrInvalidSourceEmpty = fmt.Errorf("invalid source, must specify at least one type of source configuration")

func (s Source) Validate() error {
	err := validation.Validate("source", validation.By(func(value any) error {
		if reflect.ValueOf(s).IsZero() {
			return ErrInvalidSourceEmpty
		}

		return nil
	}))
	if err != nil {
		return err
	}

	return validation.ValidateStruct(&s)
}

func (s Source) Load(ctx context.Context) ([]*SourceEntry, error) {
	if s.Local != nil {
		return s.Local.Load(ctx)
	}
	if s.Inline != nil {
		return s.Inline.Load(ctx)
	}

	return nil, ErrInvalidSourceEmpty
}
