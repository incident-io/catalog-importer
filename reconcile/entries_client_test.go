package reconcile_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/reconcile"
	"github.com/samber/lo"

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
					ArchivedEntryIds: []string{"a"},
					NotFoundEntryIds: []string{"b"},
				})).To(Succeed())
			}
		})

		It("returns an error for each entry that wasn't archived", func() {
			errs, err := entries.BulkArchive(context.Background(), "type-123", []client.CatalogEntryV3{
				{Id: "a"}, {Id: "b"}, {Id: "c"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(requested).To(Equal(client.CatalogBulkArchiveEntriesPayloadV3{
				CatalogTypeId: "type-123",
				EntryIds:      []string{"a", "b", "c"},
			}))

			Expect(errs).To(HaveLen(3))
			Expect(errs[0]).NotTo(HaveOccurred())
			Expect(errs[1]).To(MatchError(ContainSubstring("not found")))
			Expect(errs[2]).To(MatchError(ContainSubstring("missing from bulk archive response")))
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

	Describe("BulkUpsert", func() {
		var requested client.CatalogBulkUpsertEntriesPayloadV3
		BeforeEach(func() {
			handler = func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()
				Expect(r.URL.Path).To(Equal("/v3/catalog_entries/actions/bulk_upsert"))
				Expect(json.NewDecoder(r.Body).Decode(&requested)).To(Succeed())

				w.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(w).Encode(client.CatalogBulkUpsertEntriesResultV3{
					Results: []client.BulkUpsertEntryResultV3{
						{ExternalId: "a", Status: client.BulkUpsertEntryResultV3StatusCreated, CatalogEntryId: lo.ToPtr("entry-a")},
						{ExternalId: "b", Status: client.BulkUpsertEntryResultV3StatusFailed, Errors: &[]client.BulkUpsertEntryErrorV3{
							{Field: lo.ToPtr("aliases"), Message: "alias already in use"},
							{Message: "something else"},
						}},
					},
				})).To(Succeed())
			}
		})

		It("returns a result for each entry", func() {
			results, err := entries.BulkUpsert(context.Background(), "type-123", []client.CatalogCreateEntryPayloadV3{
				{CatalogTypeId: "type-123", Name: "A", ExternalId: lo.ToPtr("a"), Rank: lo.ToPtr(int32(3)), Aliases: lo.ToPtr([]string{"alias-a"})},
				{CatalogTypeId: "type-123", Name: "B", ExternalId: lo.ToPtr("b")},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(requested).To(Equal(client.CatalogBulkUpsertEntriesPayloadV3{
				CatalogTypeId: "type-123",
				Entries: []client.BulkUpsertEntryPayloadV3{
					{Name: "A", ExternalId: "a", Rank: lo.ToPtr(int32(3)), Aliases: lo.ToPtr([]string{"alias-a"})},
					{Name: "B", ExternalId: "b"},
				},
			}))

			Expect(results).To(HaveLen(2))
			Expect(results[0].Err).NotTo(HaveOccurred())
			Expect(results[0].Entry.Id).To(Equal("entry-a"))
			Expect(results[1].Err).To(MatchError("aliases: alias already in use; something else"))
		})
	})
})
