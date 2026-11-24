package reconcile_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/reconcile"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EntriesClientFromClient", func() {
	var (
		server  *httptest.Server
		handler http.HandlerFunc
		entries reconcile.EntriesClient
	)
	BeforeEach(func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler(w, r)
		}))
		DeferCleanup(server.Close)

		cl, err := client.New(context.Background(), "api-key", server.URL, "test", kitlog.NewNopLogger())
		Expect(err).NotTo(HaveOccurred())
		entries = reconcile.EntriesClientFromClient(cl)
	})

	Describe("BulkArchive", func() {
		var requested client.CatalogBulkArchiveEntriesPayloadV3
		BeforeEach(func() {
			handler = func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()
				Expect(r.URL.Path).To(Equal("/v3/catalog_entries/actions/bulk_archive"))
				Expect(json.NewDecoder(r.Body).Decode(&requested)).To(Succeed())

				w.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(w).Encode(client.CatalogBulkArchiveEntriesResultV3{
					ArchivedEntryIds:   []string{"a"},
					NotFoundEntryIds:   []string{"b"},
					NotAllowedEntryIds: []string{"c"},
				})).To(Succeed())
			}
		})

		It("returns an error for each entry that wasn't archived", func() {
			errs, err := entries.BulkArchive(context.Background(), "type-123", []client.CatalogEntryV3{
				{Id: "a"}, {Id: "b"}, {Id: "c"}, {Id: "d"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(requested).To(Equal(client.CatalogBulkArchiveEntriesPayloadV3{
				CatalogTypeId: "type-123",
				EntryIds:      []string{"a", "b", "c", "d"},
			}))

			Expect(errs).To(HaveLen(4))
			Expect(errs[0]).NotTo(HaveOccurred())
			Expect(errs[1]).To(MatchError(ContainSubstring("not found")))
			Expect(errs[2]).To(MatchError(ContainSubstring("not allowed")))
			Expect(errs[3]).To(MatchError(ContainSubstring("missing from bulk archive response")))
		})

		When("the endpoint doesn't exist", func() {
			BeforeEach(func() {
				handler = func(w http.ResponseWriter, r *http.Request) {
					http.NotFound(w, r)
				}
			})

			It("returns a not found error", func() {
				_, err := entries.BulkArchive(context.Background(), "type-123", []client.CatalogEntryV3{{Id: "a"}})
				Expect(client.IsNotFound(err)).To(BeTrue())
			})
		})
	})
})
