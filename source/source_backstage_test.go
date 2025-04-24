package source_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	kitlog "github.com/go-kit/kit/log"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/incident-io/catalog-importer/v2/source"
	"github.com/jarcoal/httpmock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SourceBackstage", func() {
	var (
		ctx    context.Context
		logger kitlog.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	})

	var (
		s      source.SourceBackstage
		client *http.Client
		mock   *httpmock.MockTransport
	)
	Context("using /api/catalog/entities", func() {
		BeforeEach(func() {
			s = source.SourceBackstage{
				Endpoint: "https://example.com/api/catalog/entities",
			}

			client = cleanhttp.DefaultClient()
			mock = httpmock.NewMockTransport()
			client.Transport = mock
		})

		Describe("Load", func() {
			var backstageRequest *http.Request

			BeforeEach(func() {
				mock.RegisterResponder(
					http.MethodGet,
					"https://example.com/api/catalog/entities",
					func(req *http.Request) (*http.Response, error) {
						backstageRequest = req
						resp, err := httpmock.NewJsonResponse(http.StatusOK, []map[string]any{})
						Expect(err).To(Succeed())
						return resp, nil
					},
				)
			})

			JustBeforeEach(func() {
				_, err := s.Load(ctx, logger, client)
				Expect(err).NotTo(HaveOccurred())
				Expect(backstageRequest).NotTo(BeNil())
			})

			Context("page size", func() {
				When("no page size is specified", func() {
					It("uses the default page size", func() {
						Expect(backstageRequest.URL.Query().Get("limit")).To(Equal("100"))
					})
				})

				When("the page size is overridden", func() {
					BeforeEach(func() {
						s.PageSize = 30
					})

					It("uses the default page size", func() {
						Expect(backstageRequest.URL.Query().Get("limit")).To(Equal("30"))
					})
				})
			})

			Context("filter", func() {
				When("no filter is specified", func() {
					It("uses the default page size", func() {
						Expect(backstageRequest.URL.Query().Has("filter")).To(BeFalse())
					})
				})

				When("a filter is specified", func() {
					BeforeEach(func() {
						s.Filter = "kind=user,metadata.namespace=default"
					})

					It("is included in the request", func() {
						Expect(backstageRequest.URL.Query().Get("filter")).To(Equal("kind=user,metadata.namespace=default"))
					})
				})
			})
		})
	})
	Context("using /api/catalog/entities/by-query", func() {
		BeforeEach(func() {
			s = source.SourceBackstage{
				Endpoint: "https://example.com/api/catalog/entities/by-query",
			}

			client = cleanhttp.DefaultClient()
			mock = httpmock.NewMockTransport()
			client.Transport = mock
		})

		Describe("Load", func() {
			var backstageRequest *http.Request

			BeforeEach(func() {
				mock.RegisterResponder(
					http.MethodGet,
					"https://example.com/api/catalog/entities/by-query",
					func(req *http.Request) (*http.Response, error) {
						backstageRequest = req
						resp, err := httpmock.NewJsonResponse(http.StatusOK, map[string]any{"items": []any{
							json.RawMessage(`{"kind":"Component","metadata":{"name":"test"}}`),
						}})
						Expect(err).To(Succeed())
						return resp, nil
					},
				)
			})

			var sourceEntries []*source.SourceEntry

			JustBeforeEach(func() {
				var err error
				sourceEntries, err = s.Load(ctx, logger, client)
				Expect(err).NotTo(HaveOccurred())
				Expect(backstageRequest).NotTo(BeNil())
			})

			It("returns the entries", func() {
				Expect(sourceEntries).To(HaveLen(1))
				Expect(sourceEntries[0].Origin).To(Equal("backstage (endpoint=https://example.com/api/catalog/entities/by-query)"))
				Expect(sourceEntries[0].Content).To(MatchJSON(`{"kind":"Component","metadata":{"name":"test"}}`))
			})

			Context("page size", func() {
				When("no page size is specified", func() {
					It("uses the default page size", func() {
						Expect(backstageRequest.URL.Query().Get("limit")).To(Equal("100"))
					})
				})

				When("the page size is overridden", func() {
					BeforeEach(func() {
						s.PageSize = 30
					})

					It("uses the default page size", func() {
						Expect(backstageRequest.URL.Query().Get("limit")).To(Equal("30"))
					})
				})
			})

			Context("filter", func() {
				When("no filter is specified", func() {
					It("uses the default page size", func() {
						Expect(backstageRequest.URL.Query().Has("filter")).To(BeFalse())
					})
				})

				When("a filter is specified", func() {
					BeforeEach(func() {
						s.Filter = "kind=user,metadata.namespace=default"
					})

					It("is included in the request", func() {
						Expect(backstageRequest.URL.Query().Get("filter")).To(Equal("kind=user,metadata.namespace=default"))
					})
				})
			})
		})
	})
})
