package source

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-ozzo/ozzo-validation/is"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/machinebox/graphql"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/stretchr/objx"
	"gopkg.in/guregu/null.v3"
)

type SourceGraphQL struct {
	Endpoint string                `json:"endpoint"` // https://api.github.com/graphql
	Headers  map[string]Credential `json:"headers"`
	Query    string                `json:"query"`
	Result   null.String           `json:"result"`
	Paginate struct {
		NextCursor null.String `json:"next_cursor"`
	} `json:"paginate,omitempty"`
}

func (s SourceGraphQL) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Endpoint,
			validation.Required.Error("must provide the GraphQL endpoint"),
			is.URL,
		),
		validation.Field(&s.Query,
			validation.Required.Error("must provide the GraphQL query"),
			validation.By(func(value any) error {
				query := value.(string)
				if s.Paginate.NextCursor.Valid {
					if !strings.Contains(query, "$cursor") {
						return fmt.Errorf("next cursor was provided but the query has no $cursor variable")
					}
				} else {
					// We're not cursor paginating, which means we need to provide either a page or
					// an offset variable.
					if !strings.Contains(query, "$page") && !strings.Contains(query, "$offset") {
						return fmt.Errorf("no pagination strategy provided by the GraphQL query")
					}
				}

				return nil
			}),
		),
	)
}

func (s SourceGraphQL) String() string {
	return fmt.Sprintf("graphql (endpoint=%s)", s.Endpoint)
}

func (s SourceGraphQL) Load(ctx context.Context, logger kitlog.Logger) ([]*SourceEntry, error) {
	client := graphql.NewClient(s.Endpoint,
		graphql.WithHTTPClient(cleanhttp.DefaultClient()))
	client.Log = func(msg string) {
		logger.Log("msg", msg)
	}

	req := graphql.NewRequest(s.Query)
	for key, value := range s.Headers {
		req.Header.Set(key, string(value))
	}

	// Infer from the query whether we should try paginating.
	shouldPaginate := s.Paginate.NextCursor.Valid ||
		strings.Contains(s.Query, "$page") ||
		strings.Contains(s.Query, "$offset")

	trySet := func(name, value string) {
		if strings.Contains(s.Query, "$"+name) {
			req.Var(name, value)
		}
	}

	var (
		page   = 0
		offset = 0
		cursor *string
	)

	entries := []*SourceEntry{}
	for {
		// Some GraphQL APIs paginate using page or offset while others use cursor.
		trySet("page", fmt.Sprintf("%d", page))
		trySet("offset", fmt.Sprintf("%d", offset))
		if cursor != nil {
			req.Var("cursor", *cursor)
		}

		var data json.RawMessage

		logger.Log("msg", "issuing GraphQL query",
			"page", page, "offset", offset, "cursor", cursor)
		err := client.Run(ctx, req, &data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to execute GraphQL query")
		}

		resp, err := objx.FromJSON(string(data))
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse GraphQL response")
		}

		result := resp.Value().Data()
		if s.Result.Valid {
			result = resp.Get(s.Result.String).Data()
		}

		if reflect.TypeOf(result).Kind() != reflect.Slice {
			return nil, errors.Wrap(err, "result is not a slice of values")
		}

		content, err := json.Marshal(result)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling result into JSON")
		}

		entries = append(entries, &SourceEntry{
			Origin:  s.String(),
			Content: content,
		})

		resultCount := reflect.ValueOf(result).Len()

		if resultCount == 0 || !shouldPaginate {
			return entries, nil
		}

		page += 1
		offset += resultCount

		// If we've configured cursor pagination then fetch the next cursor value.
		if s.Paginate.NextCursor.Valid {
			value := resp.Get(s.Paginate.NextCursor.String).Str()
			if value == "" {
				return nil, fmt.Errorf("response did not find next cursor at '%s'", s.Paginate.NextCursor.String)
			}

			cursor = lo.ToPtr(value)
		}
	}
}
