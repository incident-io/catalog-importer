package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"github.com/pkg/errors"
)

const maxRetries = 10

func attentiveBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	// Retry for rate limits and server errors.
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		// Check for a 'Retry-After' header.
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			retryAfterDate, err := time.Parse(time.RFC1123, retryAfter)
			if err != nil {
				// If we can't parse the Retry-After, lets just wait for 10 seconds
				return 10
			}

			timeToWait := time.Until(retryAfterDate)

			if timeToWait < 1*time.Second {
				// by default lets back off at least 1 second
				return 1 * time.Second
			}

			return timeToWait
		}

	}
	// otherwise use the default backoff
	return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
}

var _ retryablehttp.Logger = &retryableHttpLogger{}

type retryableHttpLogger struct {
	kitlog.Logger
}

func (l *retryableHttpLogger) Printf(message string, args ...interface{}) {
	level.Debug(l.Logger).Log("req", fmt.Sprintf(message, args...))
}

func New(ctx context.Context, apiKey, apiEndpoint, version string, logger kitlog.Logger, opts ...ClientOption) (*ClientWithResponses, error) {
	bearerTokenProvider, bearerTokenProviderErr := securityprovider.NewSecurityProviderBearerToken(apiKey)
	if bearerTokenProviderErr != nil {
		return nil, bearerTokenProviderErr
	}

	retryClient := retryablehttp.NewClient()
	retryClient.Logger = &retryableHttpLogger{logger}
	retryClient.RetryMax = maxRetries
	retryClient.Backoff = attentiveBackoff
	retryClient.HTTPClient.Transport = &http.Transport{
		MaxConnsPerHost: 10,
	}

	base := retryClient.StandardClient()

	// The generated client won't turn validation errors into actual errors, so we do this
	// inside of a generic middleware.
	base.Transport = Wrap(base.Transport, func(req *http.Request, next http.RoundTripper) (*http.Response, error) {
		resp, err := next.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if err == nil && resp.StatusCode > 299 {
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("status %d: no response body", resp.StatusCode)
			}

			return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(data))
		}

		return resp, err
	})

	clientOpts := append([]ClientOption{
		WithHTTPClient(base),
		WithRequestEditorFn(bearerTokenProvider.Intercept),
		// Add a user-agent so we can tell which version these requests came from.
		WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Add("user-agent", fmt.Sprintf("catalog-importer/%s", version))
			return nil
		}),
	}, opts...)

	client, err := NewClientWithResponses(apiEndpoint, clientOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "creating client")
	}

	return client, nil
}

// WithReadOnly restricts the client to GET requests only, useful when creating a client
// for the purpose of dry-running.
func WithReadOnly() ClientOption {
	return WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		if req.Method != http.MethodGet {
			return fmt.Errorf("read-only client tried to make mutating request: %s %s", req.Method, req.URL.String())
		}

		return nil
	})
}

// RoundTripperFunc wraps a function to implement the RoundTripper interface, allowing
// easy wrapping of existing round-trippers.
type RoundTripperFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Wrap allows easy wrapping of an existing RoundTripper with a function that can
// optionally call the original, or do its own thing.
func Wrap(next http.RoundTripper, apply func(req *http.Request, next http.RoundTripper) (*http.Response, error)) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return apply(req, next)
	})
}
