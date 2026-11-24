package reconcile

import (
	"context"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/client"
	"github.com/samber/lo"
)

type GetEntriesFunc func(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV3, []client.CatalogEntryV3, error)

// EntriesPrefetch lists the existing entries of many catalog types in the background, so
// reconciling a type only waits for whatever of its listing hasn't finished yet.
type EntriesPrefetch struct {
	getEntries GetEntriesFunc
	mu         sync.Mutex
	results    map[string]*prefetchedEntries
}

type prefetchedEntries struct {
	done        chan struct{}
	catalogType *client.CatalogTypeV3
	entries     []client.CatalogEntryV3
	err         error
}

// PrefetchEntries starts listing the entries of each catalog type, in the order given,
// with at most maxConcurrency types listing at once. Pages within a type are still listed
// one after another, as each page's cursor comes from the previous one.
func PrefetchEntries(ctx context.Context, logger kitlog.Logger, getEntries GetEntriesFunc, catalogTypes []*client.CatalogTypeV3, pageSize, maxConcurrency int) *EntriesPrefetch {
	p := &EntriesPrefetch{
		getEntries: getEntries,
		results:    map[string]*prefetchedEntries{},
	}

	catalogTypes = lo.UniqBy(catalogTypes, func(catalogType *client.CatalogTypeV3) string {
		return catalogType.Id
	})
	results := lo.Map(catalogTypes, func(catalogType *client.CatalogTypeV3, _ int) *prefetchedEntries {
		result := &prefetchedEntries{done: make(chan struct{})}
		p.results[catalogType.Id] = result
		return result
	})

	go func() {
		sem := make(chan struct{}, maxConcurrency)
		for idx, catalogType := range catalogTypes {
			result := results[idx]

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				result.err = ctx.Err()
				close(result.done)
				continue
			}

			go func() {
				defer func() { <-sem }()
				defer close(result.done)

				start := time.Now()
				result.catalogType, result.entries, result.err = getEntries(ctx, catalogType.Id, pageSize)
				if result.err == nil {
					logger.Log("msg", "listed existing entries",
						"catalog_type_id", catalogType.Id,
						"catalog_type_name", catalogType.TypeName,
						"count", len(result.entries),
						"duration", time.Since(start).Round(time.Millisecond).String(),
					)
				}
			}()
		}
	}()

	return p
}

// GetEntries returns the entries listed in the background for this catalog type, waiting
// for the listing to finish if it hasn't yet. Each listing is handed out once, so we can
// free its entries after reconciling: a later call, or a call for a type we didn't
// prefetch, lists the entries again.
func (p *EntriesPrefetch) GetEntries(ctx context.Context, catalogTypeID string, pageSize int) (*client.CatalogTypeV3, []client.CatalogEntryV3, error) {
	p.mu.Lock()
	result, ok := p.results[catalogTypeID]
	delete(p.results, catalogTypeID)
	p.mu.Unlock()

	if !ok {
		return p.getEntries(ctx, catalogTypeID, pageSize)
	}

	select {
	case <-result.done:
		return result.catalogType, result.entries, result.err
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}
