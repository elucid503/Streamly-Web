package services

import (
	"hash/fnv"
	"strings"

	mediakit "mediakit"
)

var showTrendingFallback = []string{
	"wednesday",
	"stranger things",
	"the last of us",
	"breaking bad",
	"squid game",
	"the bear",
	"house of the dragon",
	"yellowstone",
}

var movieBrowseQueries = []string{
	"2025", "action", "horror", "comedy", "drama", "thriller",
	"adventure", "sci-fi", "romance", "animation", "fantasy", "crime",
}

var showBrowseQueries = []string{
	"2024", "netflix", "drama", "hbo", "marvel", "comedy",
	"crime", "fantasy", "documentary", "reality", "anime", "british",
}

var categoryQueryOverrides = map[string]string{
	"top_dvd_streaming":                  "popular movies",
	"certified_fresh_movies":             "award winning",
	"certified_fresh_movies_on_theaters": "theater",
	"opening_this_week":                  "new release",
	"coming_soon_in_theaters":            "coming soon",
	"coming_soon":                        "upcoming",
	"new_tv_tonight":                     "new series",
	"most_popular_tv_on_rt":              "popular series",
	"certified_fresh_tv":                 "best series",
	"reelgood_treading_tv_netflix":       "netflix",
	"reelgood_treading_tv_hulu_plus":     "hulu",
	"reelgood_treading_tv_amazon":        "amazon prime",
	"reelgood_treading_tv_disney":        "disney",
}

func (s *MediaService) TrendingHits(kind mediakit.MediaKind, limit int) ([]SearchResultDTO, error) {
	snap := s.snapshot()
	if kind == mediakit.MediaMovie {
		return sliceLimit(snap.movieTrending, limit), nil
	}
	return sliceLimit(snap.showTrending, limit), nil
}

func (s *MediaService) loadTrendingHits(kind mediakit.MediaKind, limit int) ([]SearchResultDTO, error) {
	if limit <= 0 {
		limit = 12
	}

	s.throttleUpstream()
	keywords, err := s.client.Trending(kind, limit)
	if err != nil || len(keywords) == 0 {
		if kind == mediakit.MediaShow {
			keywords = append([]string(nil), showTrendingFallback...)
		} else if err != nil {
			return nil, err
		}
	}

	if len(keywords) > limit {
		keywords = keywords[:limit]
	}

	return s.resolveKeywordHits(kind, keywords, limit)
}

func (s *MediaService) loadCategoryTitles(kind mediakit.MediaKind, categoryID, categoryName string, limit int) ([]SearchResultDTO, error) {
	query := categoryBrowseQuery(kind, categoryID, categoryName)
	return s.browseByQuery(kind, query, limit, true)
}

func (s *MediaService) resolveKeywordHits(kind mediakit.MediaKind, keywords []string, limit int) ([]SearchResultDTO, error) {
	seen := make(map[int]struct{})
	out := make([]SearchResultDTO, 0, limit)

	for _, keyword := range keywords {
		if len(out) >= limit {
			break
		}

		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}

		hits, err := s.searchUpstream(keyword)
		if err != nil {
			if isRateLimitError(err) {
				return out, err
			}
			continue
		}

		for _, hit := range hits {
			if hit.Kind != kind {
				continue
			}
			if _, ok := seen[hit.ID]; ok {
				continue
			}
			seen[hit.ID] = struct{}{}
			out = append(out, hitToDTO(hit))
			if len(out) >= limit {
				break
			}
		}
	}

	return out, nil
}

func (s *MediaService) browseByQuery(kind mediakit.MediaKind, query string, limit int, allowFallback bool) ([]SearchResultDTO, error) {
	hits, err := s.searchUpstream(query)
	if err != nil {
		return nil, err
	}

	out := make([]SearchResultDTO, 0, limit)
	seen := make(map[int]struct{})

	for _, hit := range hits {
		if hit.Kind != kind {
			continue
		}
		if _, ok := seen[hit.ID]; ok {
			continue
		}
		seen[hit.ID] = struct{}{}
		out = append(out, hitToDTO(hit))
		if len(out) >= limit {
			break
		}
	}

	if len(out) > 0 || !allowFallback {
		return out, nil
	}

	fallback := browseQueriesFor(kind)
	idx := int(hashString(query)) % len(fallback)
	return s.browseByQuery(kind, fallback[idx], limit, false)
}

func categoryBrowseQuery(kind mediakit.MediaKind, categoryID, name string) string {
	if q, ok := categoryQueryOverrides[categoryID]; ok {
		return q
	}

	queries := browseQueriesFor(kind)
	idx := int(hashString(categoryID)) % len(queries)
	return queries[idx]
}

func browseQueriesFor(kind mediakit.MediaKind) []string {
	if kind == mediakit.MediaShow {
		return showBrowseQueries
	}
	return movieBrowseQueries
}

func hashString(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func (s *MediaService) loadCategories(kind mediakit.MediaKind) ([]CategoryDTO, error) {
	s.throttleUpstream()
	cats, err := s.client.TopCategories(kind)
	if err != nil {
		return nil, err
	}
	out := make([]CategoryDTO, len(cats))
	for i, cat := range cats {
		out[i] = CategoryDTO{
			ID:   cat.ID(),
			Name: cat.Name(),
			Kind: kindName(kind),
		}
	}
	return out, nil
}