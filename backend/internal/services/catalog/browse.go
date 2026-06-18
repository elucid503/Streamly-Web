package catalog

import (
	"hash/fnv"
	"strings"

	mediakit "mediakit"

	"streamly/internal/services/upstream"
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

	"top_dvd_streaming":                    "popular movies",
	"certified_fresh_movies":               "award winning",
	"certified_fresh_movies_on_theaters":   "theater",
	"opening_this_week":                    "new release",
	"coming_soon_in_theaters":              "coming soon",
	"coming_soon":                          "upcoming",
	"new_tv_tonight":                       "new series",
	"most_popular_tv_on_rt":                "popular series",
	"certified_fresh_tv":                   "best series",
	"reelgood_treading_tv_netflix":         "netflix",
	"reelgood_treading_tv_hulu_plus":       "hulu",
	"reelgood_treading_tv_amazon":          "amazon prime",
	"reelgood_treading_tv_disney":          "disney",

}

// HitToDTO converts a raw search hit into a SearchResultDTO.
func HitToDTO(hit mediakit.SearchHit) SearchResultDTO {

	return SearchResultDTO{

		ID: hit.ID,
		Kind: kindName(hit.Kind),

		Title: hit.Title,
		Year: hit.Year,

		Poster: hit.Poster,
		Description: hit.Description,
		Rating: hit.IMDBRating,

	}

}

func (c *Cache) loadTrendingHits(kind mediakit.MediaKind, limit int) ([]SearchResultDTO, error) {

	if limit <= 0 {

		limit = 12

	}

	keywords, err := c.client.Trending(kind, limit)

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

	return c.resolveKeywordHits(kind, keywords, limit)

}

func (c *Cache) loadCategoryTitles(kind mediakit.MediaKind, categoryID, categoryName string, limit int) ([]SearchResultDTO, error) {

	query := categoryBrowseQuery(kind, categoryID, categoryName)

	return c.browseByQuery(kind, query, limit, true)

}

func (c *Cache) resolveKeywordHits(kind mediakit.MediaKind, keywords []string, limit int) ([]SearchResultDTO, error) {

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

		hits, err := c.client.Search(keyword)

		if err != nil {

			if upstream.IsRateLimitError(err) {

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

			out = append(out, HitToDTO(hit))

			if len(out) >= limit {

				break

			}

		}

	}

	return out, nil

}

func (c *Cache) browseByQuery(kind mediakit.MediaKind, query string, limit int, allowFallback bool) ([]SearchResultDTO, error) {

	hits, err := c.client.Search(query)

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

		out = append(out, HitToDTO(hit))

		if len(out) >= limit {

			break

		}

	}

	if len(out) > 0 || !allowFallback {

		return out, nil

	}

	fallback := browseQueriesFor(kind)

	idx := int(hashString(query)) % len(fallback)

	return c.browseByQuery(kind, fallback[idx], limit, false)

}

func (c *Cache) loadCategories(kind mediakit.MediaKind) ([]CategoryDTO, error) {

	cats, err := c.client.TopCategories(kind)

	if err != nil {

		return nil, err

	}

	out := make([]CategoryDTO, len(cats))

	for i, cat := range cats {

		out[i] = CategoryDTO{

			ID: cat.ID(),
			Name: cat.Name(),
			Kind: kindName(kind),

		}

	}

	return out, nil

}

func (c *Cache) loadLiveChannels() ([]LiveChannelDTO, error) {

	catalog, err := c.client.LiveTV()

	if err != nil {

		return nil, err

	}

	channels := catalog.Channels()

	out := make([]LiveChannelDTO, len(channels))

	for i, ch := range channels {

		info, _ := ch.Info()

		out[i] = liveChannelFromInfo(info)

	}

	return out, nil

}

func liveChannelFromInfo(info mediakit.LiveChannelInfo) LiveChannelDTO {

	return LiveChannelDTO{

		ID: info.ID,
		DaddyID: info.DaddyID,

		Name: info.Name,
		Slug: info.Slug,
		Logo: info.Logo,

		Country: info.Country,
		Category: info.Category,

	}

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

func kindName(kind mediakit.MediaKind) string {

	if kind == mediakit.MediaMovie {

		return "movie"

	}

	return "show"

}

func hashString(s string) uint32 {

	h := fnv.New32a()

	_, _ = h.Write([]byte(s))

	return h.Sum32()

}
