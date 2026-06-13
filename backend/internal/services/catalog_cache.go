package services

import (
	"context"
	"log"
	"strings"
	"time"

	mediakit "mediakit"
)

const (
	catalogTitlesPerCategory = 24
	catalogTrendingLimit     = 10
	catalogLivePopularLimit  = 24
)

type catalogSnapshot struct {
	movieTrending       []SearchResultDTO
	showTrending        []SearchResultDTO
	movieCategories     []CategoryDTO
	showCategories      []CategoryDTO
	movieCategoryTitles map[string][]SearchResultDTO
	showCategoryTitles  map[string][]SearchResultDTO
	liveChannels        []LiveChannelDTO
	livePopular         []LiveChannelDTO
	searchIndex         []SearchResultDTO
	refreshedAt         time.Time
}

// StartCatalogCache loads a disk snapshot immediately and refreshes in the background.
func (s *MediaService) StartCatalogCache(ctx context.Context) {
	s.loadCatalogFromDisk()

	child, cancel := context.WithCancel(ctx)
	s.cacheCancel = cancel

	go func() {
		s.refreshCatalog()

		ticker := time.NewTicker(s.cfg.CatalogCacheTTL)
		defer ticker.Stop()

		for {
			select {
			case <-child.Done():
				return
			case <-ticker.C:
				s.refreshCatalog()
			}
		}
	}()
}

func (s *MediaService) StopCatalogCache() {
	if s.cacheCancel != nil {
		s.cacheCancel()
	}
}

func (s *MediaService) refreshCatalog() {
	start := time.Now()
	log.Println("[catalog-cache] refresh started")

	s.beginRefreshBatch()
	defer s.endRefreshBatch()

	next := catalogSnapshot{
		movieCategoryTitles: make(map[string][]SearchResultDTO),
		showCategoryTitles:  make(map[string][]SearchResultDTO),
	}

	var errs []string

	if movieTrending, err := retryUpstream(4, func() ([]SearchResultDTO, error) {
		return s.loadTrendingHits(mediakit.MediaMovie, catalogTrendingLimit)
	}); err != nil {
		errs = append(errs, "movie trending: "+err.Error())
		s.catalogMu.RLock()
		next.movieTrending = append([]SearchResultDTO(nil), s.catalog.movieTrending...)
		s.catalogMu.RUnlock()
	} else {
		next.movieTrending = movieTrending
	}

	if showTrending, err := retryUpstream(4, func() ([]SearchResultDTO, error) {
		return s.loadTrendingHits(mediakit.MediaShow, catalogTrendingLimit)
	}); err != nil {
		errs = append(errs, "show trending: "+err.Error())
		s.catalogMu.RLock()
		next.showTrending = append([]SearchResultDTO(nil), s.catalog.showTrending...)
		s.catalogMu.RUnlock()
	} else {
		next.showTrending = showTrending
	}

	if cats, err := retryUpstream(4, func() ([]CategoryDTO, error) {
		return s.loadCategories(mediakit.MediaMovie)
	}); err != nil {
		errs = append(errs, "movie categories: "+err.Error())
		s.catalogMu.RLock()
		next.movieCategories = append([]CategoryDTO(nil), s.catalog.movieCategories...)
		s.catalogMu.RUnlock()
	} else {
		next.movieCategories = cats
		for _, cat := range cats {
			titles, err := retryUpstream(3, func() ([]SearchResultDTO, error) {
				return s.loadCategoryTitles(mediakit.MediaMovie, cat.ID, cat.Name, catalogTitlesPerCategory)
			})
			if err != nil {
				errs = append(errs, "movie category "+cat.ID+": "+err.Error())
				s.catalogMu.RLock()
				if prev, ok := s.catalog.movieCategoryTitles[cat.ID]; ok {
					next.movieCategoryTitles[cat.ID] = append([]SearchResultDTO(nil), prev...)
				}
				s.catalogMu.RUnlock()
			} else {
				next.movieCategoryTitles[cat.ID] = titles
			}
		}
	}

	if cats, err := retryUpstream(4, func() ([]CategoryDTO, error) {
		return s.loadCategories(mediakit.MediaShow)
	}); err != nil {
		errs = append(errs, "show categories: "+err.Error())
		s.catalogMu.RLock()
		next.showCategories = append([]CategoryDTO(nil), s.catalog.showCategories...)
		s.catalogMu.RUnlock()
	} else {
		next.showCategories = cats
		for _, cat := range cats {
			titles, err := retryUpstream(3, func() ([]SearchResultDTO, error) {
				return s.loadCategoryTitles(mediakit.MediaShow, cat.ID, cat.Name, catalogTitlesPerCategory)
			})
			if err != nil {
				errs = append(errs, "show category "+cat.ID+": "+err.Error())
				s.catalogMu.RLock()
				if prev, ok := s.catalog.showCategoryTitles[cat.ID]; ok {
					next.showCategoryTitles[cat.ID] = append([]SearchResultDTO(nil), prev...)
				}
				s.catalogMu.RUnlock()
			} else {
				next.showCategoryTitles[cat.ID] = titles
			}
		}
	}

	if channels, err := retryUpstream(4, func() ([]LiveChannelDTO, error) {
		return s.loadLiveChannels()
	}); err != nil {
		errs = append(errs, "live channels: "+err.Error())
		s.catalogMu.RLock()
		next.liveChannels = append([]LiveChannelDTO(nil), s.catalog.liveChannels...)
		s.catalogMu.RUnlock()
	} else {
		next.liveChannels = channels
		next.livePopular = sliceLimit(channels, catalogLivePopularLimit)
	}

	next.searchIndex = buildSearchIndex(next)
	next.refreshedAt = time.Now()

	s.catalogMu.Lock()
	s.catalog = next
	s.catalogMu.Unlock()

	s.saveCatalogToDisk(next)

	if len(errs) > 0 {
		log.Printf("[catalog-cache] refresh finished in %s with %d errors: %v",
			time.Since(start).Round(time.Millisecond), len(errs), errs)
	} else {
		log.Printf("[catalog-cache] refresh finished in %s (%d search index entries)",
			time.Since(start).Round(time.Millisecond), len(next.searchIndex))
	}
}

func buildSearchIndex(snap catalogSnapshot) []SearchResultDTO {
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

func (s *MediaService) snapshot() catalogSnapshot {
	s.catalogMu.RLock()
	defer s.catalogMu.RUnlock()
	return s.catalog
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

func (s *MediaService) loadLiveChannels() ([]LiveChannelDTO, error) {
	s.throttleUpstream()
	catalog, err := s.client.LiveTV()
	if err != nil {
		return nil, err
	}
	channels := catalog.Channels()
	out := make([]LiveChannelDTO, len(channels))
	for i, ch := range channels {
		info, _ := ch.Info()
		out[i] = liveChannelToDTO(info)
	}
	return out, nil
}

