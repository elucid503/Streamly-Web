package services

import (
	"strings"
	"time"
)

type searchCacheEntry struct {

	results []SearchResultDTO
	expiresAt time.Time

}

func normalizeSearchQuery(query string) string {

	return strings.ToLower(strings.TrimSpace(query))

}

func (s *MediaService) getSearchCache(query string) ([]SearchResultDTO, bool) {

	s.searchCacheMu.RLock()

	defer s.searchCacheMu.RUnlock()

	entry, ok := s.searchCache[query]

	if !ok || time.Now().After(entry.expiresAt) {

		return nil, false

	}

	return append([]SearchResultDTO(nil), entry.results...), true

}

func (s *MediaService) setSearchCache(query string, results []SearchResultDTO) {

	s.searchCacheMu.Lock()

	defer s.searchCacheMu.Unlock()

	ttl := s.cfg.CatalogCacheTTL

	if ttl <= 0 {

		ttl = time.Hour

	}

	s.searchCache[query] = searchCacheEntry{

		results: append([]SearchResultDTO(nil), results...),
		expiresAt: time.Now().Add(ttl),

	}

}

func (s *MediaService) Search(query string) ([]SearchResultDTO, error) {

	key := normalizeSearchQuery(query)

	if key == "" {

		return []SearchResultDTO{}, nil

	}

	if cached, ok := s.getSearchCache(key); ok {

		return cached, nil

	}

	seen := make(map[int]struct{})

	results := make([]SearchResultDTO, 0, 50)

	hits, apiErr := s.searchUpstream(key)

	if apiErr == nil {

		for _, hit := range hits {

			if _, ok := seen[hit.ID]; ok {

				continue

			}

			seen[hit.ID] = struct{}{}

			results = append(results, hitToDTO(hit))

		}

	}

	for _, hit := range filterSearchIndexWords(s.snapshot().searchIndex, key, 50) {

		if _, ok := seen[hit.ID]; ok {

			continue

		}

		seen[hit.ID] = struct{}{}

		results = append(results, hit)

	}

	if apiErr != nil && len(results) == 0 {

		return nil, apiErr

	}

	s.setSearchCache(key, results)

	return results, nil

}

func filterSearchIndexWords(index []SearchResultDTO, query string, limit int) []SearchResultDTO {

	words := strings.Fields(query)

	if len(words) == 0 {

		return nil

	}

	out := make([]SearchResultDTO, 0, limit)

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
