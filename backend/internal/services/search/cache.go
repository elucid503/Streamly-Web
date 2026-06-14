package search

import (
	"strings"
	"sync"
	"time"

	mediakit "mediakit"

	"streamly/internal/services/catalog"
	"streamly/internal/services/upstream"

	"golang.org/x/sync/singleflight"
)

const maxEntries = 256

type cacheEntry struct {

	results []catalog.SearchResultDTO
	expiresAt time.Time

}

// Cache caches upstream search results and supplements them with the catalog index.
type Cache struct {

	client *mediakit.Client
	throttle *upstream.Throttle
	catalogSnap func() catalog.Snapshot
	ttl time.Duration

	mu sync.RWMutex
	entries map[string]cacheEntry
	group singleflight.Group

}

// New builds a Cache. catalogSnap is called to read the current catalog snapshot for
// offline index filtering; ttl controls how long search results are cached.
func New(client *mediakit.Client, throttle *upstream.Throttle, catalogSnap func() catalog.Snapshot, ttl time.Duration) *Cache {

	return &Cache{

		client: client,
		throttle: throttle,
		catalogSnap: catalogSnap,
		ttl: ttl,

		entries: make(map[string]cacheEntry),

	}

}

// Search returns results for query, merging upstream hits with the catalog index.
func (c *Cache) Search(query string) ([]catalog.SearchResultDTO, error) {

	key := normalizeQuery(query)

	if key == "" {

		return []catalog.SearchResultDTO{}, nil

	}

	if cached, ok := c.get(key); ok {

		return cached, nil

	}

	result, err, _ := c.group.Do(key, func() (any, error) {

		seen := make(map[int]struct{})

		results := make([]catalog.SearchResultDTO, 0, 50)

		hits, apiErr := c.throttle.Search(c.client, key)

		if apiErr == nil {

			for _, hit := range hits {

				if _, ok := seen[hit.ID]; ok {

					continue

				}

				seen[hit.ID] = struct{}{}

				results = append(results, catalog.HitToDTO(hit))

			}

		}

		for _, hit := range filterIndexWords(c.catalogSnap().SearchIndex(), key, 50) {

			if _, ok := seen[hit.ID]; ok {

				continue

			}

			seen[hit.ID] = struct{}{}

			results = append(results, hit)

		}

		if apiErr != nil && len(results) == 0 {

			return nil, apiErr

		}

		c.set(key, results)

		return results, nil

	})

	if err != nil {

		return nil, err

	}

	return result.([]catalog.SearchResultDTO), nil

}

func (c *Cache) get(key string) ([]catalog.SearchResultDTO, bool) {

	c.mu.RLock()

	defer c.mu.RUnlock()

	entry, ok := c.entries[key]

	if !ok || time.Now().After(entry.expiresAt) {

		return nil, false

	}

	return append([]catalog.SearchResultDTO(nil), entry.results...), true

}

func (c *Cache) set(key string, results []catalog.SearchResultDTO) {

	c.mu.Lock()

	defer c.mu.Unlock()

	ttl := c.ttl

	if ttl <= 0 {

		ttl = time.Hour

	}

	c.entries[key] = cacheEntry{

		results: append([]catalog.SearchResultDTO(nil), results...),
		expiresAt: time.Now().Add(ttl),

	}

	c.pruneLocked()

}

func (c *Cache) pruneLocked() {

	now := time.Now()

	for key, entry := range c.entries {

		if now.After(entry.expiresAt) || len(c.entries) > maxEntries {

			delete(c.entries, key)

		}

	}

}

func normalizeQuery(query string) string {

	return strings.ToLower(strings.TrimSpace(query))

}

func filterIndexWords(index []catalog.SearchResultDTO, query string, limit int) []catalog.SearchResultDTO {

	words := strings.Fields(query)

	if len(words) == 0 {

		return nil

	}

	out := make([]catalog.SearchResultDTO, 0, limit)

	for _, hit := range index {

		title := strings.ToLower(hit.Title)

		matched := true

		for _, word := range words {

			if !strings.Contains(title, word) {

				matched = false

				break

			}

		}

		if !matched {

			continue

		}

		out = append(out, hit)

		if limit > 0 && len(out) >= limit {

			break

		}

	}

	return out

}
