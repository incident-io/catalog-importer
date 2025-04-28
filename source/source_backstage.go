package source

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-ozzo/ozzo-validation/is"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

type SourceBackstage struct {
	Endpoint string     `json:"endpoint"` // https://backstage.company.io/api/catalog/entities
	Token    Credential `json:"token"`
	SignJWT  *bool      `json:"sign_jwt"`
	Header   string     `json:"header"`
	PageSize int        `json:"page_size"`
	Filter   string     `json:"filter"`
}

func (s SourceBackstage) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Endpoint,
			validation.Required.Error("must provide an endpoint for fetching Backstage entries"),
			is.URL,
		),
	)
}

func (s SourceBackstage) String() string {
	return fmt.Sprintf("backstage (endpoint=%s)", s.Endpoint)
}

func (s SourceBackstage) Load(ctx context.Context, logger kitlog.Logger, client *http.Client) ([]*SourceEntry, error) {
	token, err := s.getToken()
	if err != nil {
		return nil, errors.Wrap(err, "getting Backstage token")
	}

	endpointURL, err := url.Parse(s.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "parsing Backstage URL")
	}

	if endpointURL.Path == "/api/catalog/entities" {
		logger.Log(
			"msg",
			"Deprecated: The Backstage API endpoint '/api/catalog/entities' is deprecated. Use '/api/catalog/entities/by-query' instead.",
		)
		return s.fetchEntries(ctx, client, token)
	}
	if endpointURL.Path == "/api/catalog/entities/by-query" {
		return s.fetchEntriesByQuery(ctx, client, token)
	}

	return nil, errors.New("Backstage endpoint must have path of '/api/catalog/entities' or '/api/catalog/entities/by-query'")
}

const defaultPageSize = 100

// https://backstage.io/docs/features/software-catalog/software-catalog-api/#get-entities
func (s SourceBackstage) fetchEntries(ctx context.Context, client *http.Client, token string) ([]*SourceEntry, error) {
	var (
		limit  = defaultPageSize
		offset = 0
	)

	if s.PageSize != 0 {
		limit = s.PageSize
	}

	entries := []*SourceEntry{}
	for {
		query := url.Values{}
		query.Set("limit", strconv.Itoa(limit))
		query.Set("offset", strconv.Itoa(offset))

		if s.Filter != "" {
			query.Set("filter", s.Filter)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Endpoint+"?"+query.Encode(), nil)
		if err != nil {
			return nil, errors.Wrap(err, "building Backstage URL")
		}

		if token != "" {

			header := s.Header

			if header == "" {
				header = "Authorization"
			}

			req.Header.Add(header, fmt.Sprintf("Bearer %s", token))
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("received error from Backstage: %s", resp.Status)
		}
		if err != nil {
			return nil, errors.Wrap(err, "fetching Backstage entries")
		}

		page := []json.RawMessage{}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			return nil, errors.Wrap(err, "parsing Backstage entries")
		}

		if len(page) == 0 {
			return entries, nil
		}

		for idx := range page {
			entries = append(entries, &SourceEntry{
				Origin:  s.String(),
				Content: page[idx],
			})
		}

		offset += len(page)
	}
}

type getEntitiesByQueryResponse struct {
	Items      []json.RawMessage `json:"items"`
	TotalItems int               `json:"totalItems"`
	PageInfo   struct {
		NextCursor string `json:"nextCursor"`
	} `json:"pageInfo"`
}

// https://backstage.io/docs/features/software-catalog/software-catalog-api/#get-entitiesby-query
func (s SourceBackstage) fetchEntriesByQuery(ctx context.Context, client *http.Client, token string) ([]*SourceEntry, error) {
	var (
		limit  = defaultPageSize
		cursor = ""
	)

	if s.PageSize != 0 {
		limit = s.PageSize
	}

	entries := []*SourceEntry{}
	for {
		query := url.Values{}
		query.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			query.Set("cursor", cursor)
		}

		if s.Filter != "" {
			query.Set("filter", s.Filter)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Endpoint+"?"+query.Encode(), nil)
		if err != nil {
			return nil, errors.Wrap(err, "building Backstage URL")
		}

		if token != "" {

			header := s.Header

			if header == "" {
				header = "Authorization"
			}

			req.Header.Add(header, fmt.Sprintf("Bearer %s", token))
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("received error from Backstage: %s", resp.Status)
		}
		if err != nil {
			return nil, errors.Wrap(err, "fetching Backstage entries")
		}

		page := getEntitiesByQueryResponse{}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			return nil, errors.Wrap(err, "parsing Backstage entries")
		}

		if len(page.Items) == 0 {
			return entries, nil
		}

		for _, item := range page.Items {
			entries = append(entries, &SourceEntry{
				Origin:  s.String(),
				Content: item,
			})
		}

		if page.PageInfo.NextCursor == "" {
			return entries, nil
		}
		cursor = page.PageInfo.NextCursor
	}
}

func (s SourceBackstage) getToken() (string, error) {
	if s.Token == "" {
		// it's valid to not provide a token, in which case we just return an empty string
		return "", nil
	}

	// If not provided or explicitly enabled, sign the token into a JWT and use that as
	// the Authorization header.
	if s.SignJWT == nil || *s.SignJWT {
		return s.getJWT()
	}

	// Otherwise if someone has told us not to, don't sign the token and use it as-is.
	return string(s.Token), nil
}

// getJWT applies the rules from the Backstage docs to generate a JWT that is valid for
// external Backstage authentication.
//
// https://backstage.io/docs/auth/service-to-service-auth/#usage-in-external-callers
func (s SourceBackstage) getJWT() (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = jwt.MapClaims{
		"sub": "backstage-server",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	secret, err := base64.StdEncoding.DecodeString(string(s.Token))
	if err != nil {
		return "", errors.Wrap(err, "supplied backstage token must be a base64 string")
	}

	return token.SignedString(secret)
}
