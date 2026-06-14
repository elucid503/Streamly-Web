package catalog

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	mediakit "mediakit"

	"streamly/internal/services/upstream"
)

const (
	titlesPerCategory = 24
	trendingLimit     = 10
	livePopularLimit  = 24
)

// Cache maintains a periodically refreshed in-memory catalog snapshot.
type Cache struct {
	client   *mediakit.Client
	throttle *upstream.Throttle

	cacheTTL  time.Duration
	cacheFile string

	mu     sync.RWMutex
	snap   Snapshot
	cancel context.CancelFunc
}

// New builds a Cache. cacheTTL controls the refresh interval; cacheFile is the
// optional path for disk persistence (empty disables disk caching).
func New(client *mediakit.Client, throttle *upstream.Throttle, cacheTTL time.Duration, cacheFile string) *Cache {

	return &Cache{

		client:   client,
		throttle: throttle,

		cacheTTL:  cacheTTL,
		cacheFile: cacheFile,

		snap: Snapshot{

			movieCategoryTitles: make(map[string][]SearchResultDTO),
			showCategoryTitles:  make(map[string][]SearchResultDTO),
		},
	}

}

// Start loads the disk snapshot immediately then refreshes on the configured interval.
func (c *Cache) Start(ctx context.Context) {

	c.loadFromDisk()

	child, cancel := context.WithCancel(ctx)

	c.cancel = cancel

	go func() {

		c.refresh()

		ttl := c.cacheTTL

		if ttl <= 0 {

			ttl = time.Hour

		}

		ticker := time.NewTicker(ttl)

		defer ticker.Stop()

		for {

			select {

			case <-child.Done():
				return
			case <-ticker.C:
				c.refresh()

			}

		}

	}()

}

// Stop cancels the background refresh goroutine.
func (c *Cache) Stop() {

	if c.cancel != nil {

		c.cancel()

	}

}

// Snapshot returns an immutable copy of the current catalog state.
func (c *Cache) Snapshot() Snapshot {

	c.mu.RLock()

	defer c.mu.RUnlock()

	return c.snap

}

// TrendingHits returns the cached trending hits for the given kind.
func (c *Cache) TrendingHits(kind mediakit.MediaKind, limit int) []SearchResultDTO {

	snap := c.Snapshot()

	if kind == mediakit.MediaMovie {

		return sliceLimit(snap.movieTrending, limit)

	}

	return sliceLimit(snap.showTrending, limit)

}

// Categories returns the cached browse categories for the given kind.
func (c *Cache) Categories(kind mediakit.MediaKind) []CategoryDTO {

	snap := c.Snapshot()

	if kind == mediakit.MediaMovie {

		return append([]CategoryDTO(nil), snap.movieCategories...)

	}

	return append([]CategoryDTO(nil), snap.showCategories...)

}

// CategoryTitles returns a page of cached titles for a specific category.
func (c *Cache) CategoryTitles(kind mediakit.MediaKind, categoryID string, page, limit int) []SearchResultDTO {

	snap := c.Snapshot()

	var titles []SearchResultDTO

	if kind == mediakit.MediaMovie {

		titles = snap.movieCategoryTitles[categoryID]

	} else {

		titles = snap.showCategoryTitles[categoryID]

	}

	return slicePage(titles, page, limit)

}

// LiveChannels returns all cached live TV channels.
func (c *Cache) LiveChannels() []LiveChannelDTO {

	snap := c.Snapshot()

	return append([]LiveChannelDTO(nil), snap.liveChannels...)

}

// LivePopular returns the top limit live TV channels.
func (c *Cache) LivePopular(limit int) []LiveChannelDTO {

	snap := c.Snapshot()

	return sliceLimit(snap.livePopular, limit)

}

// LiveSearch filters live channels by name or slug.
func (c *Cache) LiveSearch(query string, limit int) []LiveChannelDTO {

	snap := c.Snapshot()

	return filterLiveChannels(snap.liveChannels, query, limit)

}

func (c *Cache) LiveChannel(daddyID string) (LiveChannelDTO, bool) {

	snap := c.Snapshot()

	for _, channel := range snap.liveChannels {

		if channel.DaddyID == daddyID || channel.ID == daddyID {

			return channel, true

		}

	}

	return LiveChannelDTO{}, false

}

func (c *Cache) refresh() {

	start := time.Now()

	log.Println("[catalog-cache] refresh started")

	c.throttle.Begin()

	defer c.throttle.End()

	next := Snapshot{

		movieCategoryTitles: make(map[string][]SearchResultDTO),
		showCategoryTitles:  make(map[string][]SearchResultDTO),
	}

	var errs []string

	if movieTrending, err := upstream.Retry(4, func() ([]SearchResultDTO, error) {

		return c.loadTrendingHits(mediakit.MediaMovie, trendingLimit)

	}); err != nil {

		errs = append(errs, "movie trending: "+err.Error())

		c.mu.RLock()

		next.movieTrending = append([]SearchResultDTO(nil), c.snap.movieTrending...)

		c.mu.RUnlock()

	} else {

		next.movieTrending = movieTrending

	}

	if showTrending, err := upstream.Retry(4, func() ([]SearchResultDTO, error) {

		return c.loadTrendingHits(mediakit.MediaShow, trendingLimit)

	}); err != nil {

		errs = append(errs, "show trending: "+err.Error())

		c.mu.RLock()

		next.showTrending = append([]SearchResultDTO(nil), c.snap.showTrending...)

		c.mu.RUnlock()

	} else {

		next.showTrending = showTrending

	}

	if cats, err := upstream.Retry(4, func() ([]CategoryDTO, error) {

		return c.loadCategories(mediakit.MediaMovie)

	}); err != nil {

		errs = append(errs, "movie categories: "+err.Error())

		c.mu.RLock()

		next.movieCategories = append([]CategoryDTO(nil), c.snap.movieCategories...)

		c.mu.RUnlock()

	} else {

		next.movieCategories = cats

		for _, cat := range cats {

			titles, err := upstream.Retry(3, func() ([]SearchResultDTO, error) {

				return c.loadCategoryTitles(mediakit.MediaMovie, cat.ID, cat.Name, titlesPerCategory)

			})

			if err != nil {

				errs = append(errs, "movie category "+cat.ID+": "+err.Error())

				c.mu.RLock()

				if prev, ok := c.snap.movieCategoryTitles[cat.ID]; ok {

					next.movieCategoryTitles[cat.ID] = append([]SearchResultDTO(nil), prev...)

				}

				c.mu.RUnlock()

			} else {

				next.movieCategoryTitles[cat.ID] = titles

			}

		}

	}

	if cats, err := upstream.Retry(4, func() ([]CategoryDTO, error) {

		return c.loadCategories(mediakit.MediaShow)

	}); err != nil {

		errs = append(errs, "show categories: "+err.Error())

		c.mu.RLock()

		next.showCategories = append([]CategoryDTO(nil), c.snap.showCategories...)

		c.mu.RUnlock()

	} else {

		next.showCategories = cats

		for _, cat := range cats {

			titles, err := upstream.Retry(3, func() ([]SearchResultDTO, error) {

				return c.loadCategoryTitles(mediakit.MediaShow, cat.ID, cat.Name, titlesPerCategory)

			})

			if err != nil {

				errs = append(errs, "show category "+cat.ID+": "+err.Error())

				c.mu.RLock()

				if prev, ok := c.snap.showCategoryTitles[cat.ID]; ok {

					next.showCategoryTitles[cat.ID] = append([]SearchResultDTO(nil), prev...)

				}

				c.mu.RUnlock()

			} else {

				next.showCategoryTitles[cat.ID] = titles

			}

		}

	}

	if channels, err := upstream.Retry(4, func() ([]LiveChannelDTO, error) {

		return c.loadLiveChannels()

	}); err != nil {

		errs = append(errs, "live channels: "+err.Error())

		c.mu.RLock()

		next.liveChannels = append([]LiveChannelDTO(nil), c.snap.liveChannels...)

		c.mu.RUnlock()

	} else {

		next.liveChannels = channels

		next.livePopular = sliceLimit(channels, livePopularLimit)

	}

	next.searchIndex = buildSearchIndex(next)

	next.refreshedAt = time.Now()

	c.mu.Lock()

	c.snap = next

	c.mu.Unlock()

	c.saveToDisk(next)

	if len(errs) > 0 {

		log.Printf("[catalog-cache] refresh finished in %s with %d errors: %v",
			time.Since(start).Round(time.Millisecond), len(errs), errs)

	} else {

		log.Printf("[catalog-cache] refresh finished in %s (%d search index entries)",
			time.Since(start).Round(time.Millisecond), len(next.searchIndex))

	}

}

func buildSearchIndex(snap Snapshot) []SearchResultDTO {

	seen := make(map[int]struct{})

	var out []SearchResultDTO

	add := func(items []SearchResultDTO) {

		for _, item := range items {

			if _, ok := seen[item.ID]; ok {

				continue

			}

			seen[item.ID] = struct{}{}

			out = append(out, item)

		}

	}

	add(snap.movieTrending)

	add(snap.showTrending)

	for _, titles := range snap.movieCategoryTitles {

		add(titles)

	}

	for _, titles := range snap.showCategoryTitles {

		add(titles)

	}

	return out

}

func slicePage[T any](items []T, page, limit int) []T {

	if page <= 0 {

		page = 1

	}

	if limit <= 0 {

		limit = len(items)

	}

	start := (page - 1) * limit

	if start >= len(items) {

		return []T{}

	}

	end := start + limit

	if end > len(items) {

		end = len(items)

	}

	return append([]T(nil), items[start:end]...)

}

func sliceLimit[T any](items []T, limit int) []T {

	if limit <= 0 || limit >= len(items) {

		return append([]T(nil), items...)

	}

	return append([]T(nil), items[:limit]...)

}

func filterSearchIndex(index []SearchResultDTO, query string, limit int) []SearchResultDTO {

	query = strings.ToLower(strings.TrimSpace(query))

	if query == "" {

		return nil

	}

	out := make([]SearchResultDTO, 0, limit)

	for _, hit := range index {

		if !strings.Contains(strings.ToLower(hit.Title), query) {

			continue

		}

		out = append(out, hit)

		if limit > 0 && len(out) >= limit {

			break

		}

	}

	return out

}

func filterLiveChannels(channels []LiveChannelDTO, query string, limit int) []LiveChannelDTO {

	query = strings.ToLower(strings.TrimSpace(query))

	if query == "" {

		return sliceLimit(channels, limit)

	}

	out := make([]LiveChannelDTO, 0, limit)

	for _, ch := range channels {

		name := strings.ToLower(ch.Name)

		slug := strings.ToLower(ch.Slug)

		if !strings.Contains(name, query) && !strings.Contains(slug, query) {

			continue

		}

		out = append(out, ch)

		if limit > 0 && len(out) >= limit {

			break

		}

	}

	return out

}
