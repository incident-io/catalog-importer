package source

import (
	"context"
	"fmt"
	"reflect"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// SourceEntry is an entry that has been discovered in a source, with the contents of the
// source file and an Origin that explains where the entry came from, specific to the type
// of source that produced it.
type SourceEntry struct {
	Origin  string
	Content []byte
}

func (e SourceEntry) Parse() ([]Entry, error) {
	return Parse(e.Content), nil
}

// Source is instantiated from configuration and represents a source of catalog files.
type Source struct {
	Local  *SourceLocal  `json:"local,omitempty"`
	Inline *SourceInline `json:"inline,omitempty"`
	Exec   *SourceExec   `json:"exec,omitempty"`
}

func (s Source) Name() string {
	if s.Local != nil {
		return "local"
	}
	if s.Inline != nil {
		return "inline"
	}
	if s.Exec != nil {
		return "exec"
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
	if s.Exec != nil {
		return s.Exec.Load(ctx)
	}

	return nil, ErrInvalidSourceEmpty
}
