package source

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/hashicorp/go-cleanhttp"
)

// SourceEntry is an entry that has been discovered in a source, with the contents of the
// source file and an Origin that explains where the entry came from, specific to the type
// of source that produced it.
type SourceEntry struct {
	Origin   string // the source origin e.g. inline
	Filename string // the filename that it should be evaluated under e.g. app/main.jsonnet
	Content  []byte // the content of the source
}

func (e SourceEntry) Parse() ([]Entry, error) {
	entries := Parse(e.Filename, e.Content)
	if len(entries) == 0 {
		return entries, fmt.Errorf("failed to parse any entries")
	}

	return entries, nil
}

// Source is instantiated from configuration and represents a source of catalog files.
type Source struct {
	Local     *SourceLocal     `json:"local,omitempty"`
	Inline    *SourceInline    `json:"inline,omitempty"`
	Exec      *SourceExec      `json:"exec,omitempty"`
	Backstage *SourceBackstage `json:"backstage,omitempty"`
	GitHub    *SourceGitHub    `json:"github,omitempty"`
	GraphQL   *SourceGraphQL   `json:"graphql,omitempty"`
}

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

type SourceBackend interface {
	String() string
	Load(ctx context.Context, logger kitlog.Logger, client *http.Client) ([]*SourceEntry, error)
}

func (s Source) Backend() (SourceBackend, error) {
	if s.Local != nil {
		return s.Local, nil
	}
	if s.Inline != nil {
		return s.Inline, nil
	}
	if s.Exec != nil {
		return s.Exec, nil
	}
	if s.Backstage != nil {
		return s.Backstage, nil
	}
	if s.GitHub != nil {
		return s.GitHub, nil
	}
	if s.GraphQL != nil {
		return s.GraphQL, nil
	}

	return nil, ErrInvalidSourceEmpty
}

var ErrInvalidSourceEmpty = fmt.Errorf("invalid source, must specify at least one type of source configuration")

// httpClient is the client shared by every source that makes HTTP requests, so that
// connections to the same host are pooled and reused across requests. Note that the
// GitHub source builds its own client via oauth2.NewClient and so does not use this.
//
// This deliberately uses cleanhttp.DefaultPooledTransport rather than
// cleanhttp.DefaultClient: the latter sets DisableKeepAlives, which meant each request
// opened a fresh TCP connection. Paginating a large Backstage catalog then burned one
// connection (and one NAT source port) per page, which could exhaust the SNAT port pool.
var httpClient = sync.OnceValue(func() *http.Client {
	transport := cleanhttp.DefaultPooledTransport()

	// Expire idle connections below the idle timeout of anything likely to sit between us
	// and the source (ALB 60s, nginx 75s, many proxies 30-60s), so we don't pull a
	// silently reaped connection out of the pool.
	transport.IdleConnTimeout = 30 * time.Second

	return &http.Client{Transport: transport}
})

func (s Source) Load(ctx context.Context, logger kitlog.Logger) ([]*SourceEntry, error) {
	source, err := s.Backend()
	if err != nil {
		return nil, err
	}

	return source.Load(ctx, logger, httpClient())
}
