package reconcile_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/incident-io/catalog-importer/v2/reconcile"
	"github.com/samber/lo"
	"golang.org/x/time/rate"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PrefetchEntries", func() {
	var (
		ctx    context.Context
		logger kitlog.Logger

		mu      sync.Mutex
		listed  []string // catalog type IDs, in the order we started listing them
		release chan struct{}
		listErr error
	)
	BeforeEach(func() {
		ctx = context.Background()
		logger = kitlog.NewNopLogger()

		mu.Lock()
		defer mu.Unlock()
		listed = nil
		release = make(chan struct{})
		close(release) // listings finish straight away, unless a test holds them
		listErr = nil
	})
	holdListings := func() (releaseListings func()) {
		mu.Lock()
		defer mu.Unlock()
		held := make(chan struct{})
		release = held
		return func() { close(held) }
	}

	getEntries := func(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV3, []client.CatalogEntryV3, error) {
		mu.Lock()
		listed = append(listed, catalogTypeID)
		release, listErr, entryID := release, listErr, fmt.Sprintf("%s-entry-%d", catalogTypeID, len(listed))
		mu.Unlock()

		<-release
		if listErr != nil {
			return nil, nil, listErr
		}

		return &client.CatalogTypeV3{Id: catalogTypeID}, []client.CatalogEntryV3{{Id: entryID}}, nil
	}
	catalogTypes := func(ids ...string) []*client.CatalogTypeV3 {
		return lo.Map(ids, func(id string, _ int) *client.CatalogTypeV3 {
			return &client.CatalogTypeV3{Id: id, TypeName: id}
		})
	}
	listedTypes := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string{}, listed...)
	}

	It("hands back what it listed in the background, listing each type once", func() {
		prefetch := reconcile.PrefetchEntries(ctx, logger, getEntries, catalogTypes("A", "B", "A"), 250, 4)
		Eventually(listedTypes).Should(ConsistOf("A", "B"))

		for _, id := range []string{"A", "B"} {
			catalogType, entries, err := prefetch.GetEntries(ctx, id, 250)
			Expect(err).NotTo(HaveOccurred())
			Expect(catalogType.Id).To(Equal(id))
			Expect(entries).To(HaveLen(1))
		}
		Expect(listedTypes()).To(ConsistOf("A", "B"))
	})

	It("lists types in the order given", func() {
		prefetch := reconcile.PrefetchEntries(ctx, logger, getEntries, catalogTypes("A", "B", "C"), 250, 1)
		_, _, err := prefetch.GetEntries(ctx, "C", 250)
		Expect(err).NotTo(HaveOccurred())
		Expect(listedTypes()).To(Equal([]string{"A", "B", "C"}))
	})

	It("lists at most maxConcurrency types at once", func() {
		releaseListings := holdListings()
		prefetch := reconcile.PrefetchEntries(ctx, logger, getEntries, catalogTypes("A", "B", "C", "D"), 250, 2)

		Eventually(listedTypes).Should(ConsistOf("A", "B"))
		Consistently(listedTypes, 50*time.Millisecond).Should(HaveLen(2))

		releaseListings()
		for _, id := range []string{"C", "D"} {
			_, _, err := prefetch.GetEntries(ctx, id, 250)
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(listedTypes()[2:]).To(ConsistOf("C", "D"))
	})

	It("waits for a listing that hasn't finished yet", func() {
		releaseListings := holdListings()
		prefetch := reconcile.PrefetchEntries(ctx, logger, getEntries, catalogTypes("A"), 250, 4)

		got := make(chan []client.CatalogEntryV3)
		go func() {
			defer GinkgoRecover()
			_, entries, err := prefetch.GetEntries(ctx, "A", 250)
			Expect(err).NotTo(HaveOccurred())
			got <- entries
		}()
		Consistently(got, 50*time.Millisecond).ShouldNot(Receive())

		releaseListings()
		Eventually(got).Should(Receive(HaveLen(1)))
	})

	It("lists again for a type it has already handed out, or didn't prefetch", func() {
		prefetch := reconcile.PrefetchEntries(ctx, logger, getEntries, catalogTypes("A"), 250, 4)
		_, _, err := prefetch.GetEntries(ctx, "A", 250)
		Expect(err).NotTo(HaveOccurred())

		_, _, err = prefetch.GetEntries(ctx, "A", 250)
		Expect(err).NotTo(HaveOccurred())
		_, _, err = prefetch.GetEntries(ctx, "B", 250)
		Expect(err).NotTo(HaveOccurred())

		Expect(listedTypes()).To(Equal([]string{"A", "A", "B"}))
	})

	It("returns the error from listing", func() {
		mu.Lock()
		listErr = errors.New("boom")
		mu.Unlock()
		prefetch := reconcile.PrefetchEntries(ctx, logger, getEntries, catalogTypes("A"), 250, 4)

		_, _, err := prefetch.GetEntries(ctx, "A", 250)
		Expect(err).To(MatchError("boom"))
	})

	It("stops waiting when the caller's context is cancelled", func() {
		defer holdListings()()
		prefetch := reconcile.PrefetchEntries(ctx, logger, getEntries, catalogTypes("A"), 250, 4)

		callerCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, _, err := prefetch.GetEntries(callerCtx, "A", 250)
		Expect(err).To(MatchError(context.Canceled))
	})
})

var _ = Describe("GetEntries", func() {
	var (
		server   *httptest.Server
		cl       *client.ClientWithResponses
		mu       sync.Mutex
		cursors  []string
		requests []time.Time
	)
	BeforeEach(func() {
		cursors, requests = nil, nil

		// Three pages of two entries, then an empty page.
		allEntries := lo.Times(6, func(idx int) client.CatalogEntryV3 {
			return client.CatalogEntryV3{Id: fmt.Sprintf("entry-%d", idx)}
		})
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			after := r.URL.Query().Get("after")
			mu.Lock()
			cursors = append(cursors, after)
			requests = append(requests, time.Now())
			mu.Unlock()

			start := 0
			if after != "" {
				start = lo.IndexOf(lo.Map(allEntries, func(entry client.CatalogEntryV3, _ int) string { return entry.Id }), after) + 1
			}
			page := allEntries[start:min(start+2, len(allEntries))]

			w.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(w).Encode(map[string]any{
				"catalog_type":    client.CatalogTypeV3{Id: "type"},
				"catalog_entries": page,
				"pagination_meta": map[string]any{"page_size": 2},
			})).To(Succeed())
		}))
		DeferCleanup(server.Close)

		var err error
		cl, err = client.New(context.Background(), "key", server.URL, "test", kitlog.NewNopLogger())
		Expect(err).NotTo(HaveOccurred())
	})

	It("lists every page in order, each waiting on the limiter", func() {
		limiter := rate.NewLimiter(rate.Every(50*time.Millisecond), 1)
		catalogType, entries, err := reconcile.GetEntries(context.Background(), cl, "type", 2, limiter)
		Expect(err).NotTo(HaveOccurred())
		Expect(catalogType.Id).To(Equal("type"))
		Expect(lo.Map(entries, func(entry client.CatalogEntryV3, _ int) string { return entry.Id })).To(Equal([]string{
			"entry-0", "entry-1", "entry-2", "entry-3", "entry-4", "entry-5",
		}))

		Expect(cursors).To(Equal([]string{"", "entry-1", "entry-3", "entry-5"}))
		Expect(requests[len(requests)-1].Sub(requests[0])).To(BeNumerically(">=", 140*time.Millisecond))
	})
})
