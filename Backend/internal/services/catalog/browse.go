package catalog

import (
	"hash/fnv"
	"sort"
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

// HitToDTO converts a raw search hit into a SearchResultDTO.
func HitToDTO(hit mediakit.SearchHit) SearchResultDTO {

	return SearchResultDTO{

		ID:   hit.ID,
		Kind: kindName(hit.Kind),

		Title: hit.Title,
		Year:  hit.Year,

		Poster:      hit.Poster,
		Description: hit.Description,
		Rating:      hit.IMDBRating,
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

			ID:   cat.ID(),
			Name: cat.Name(),
			Kind: kindName(kind),
		}

	}

	return out, nil

}

func (c *Cache) loadLiveChannels() ([]LiveChannelDTO, error) {

	catalog, err := c.client.Channels()

	if err != nil {

		return nil, err

	}

	sorted := catalog.Sorted()

	out := make([]LiveChannelDTO, len(sorted))

	for i, ch := range sorted {

		out[i] = liveChannelFromChannel(ch)

	}

	return out, nil

}

func liveChannelFromChannel(ch mediakit.Channel) LiveChannelDTO {

	return LiveChannelDTO{

		ID: ch.ID,
		Name: ch.Name,
		Slug: ch.Slug,
		Code: ch.Code,
		Logo: ch.Logo,

		Country: ch.Country.Code,
		CountryName: ch.Country.Name,
		Category: ch.Category,
		Categories: append([]string(nil), ch.Categories...),
		Network: ch.Network,
		Owners: append([]string(nil), ch.Owners...),
		Website: ch.Website,
		AltNames: append([]string(nil), ch.AltNames...),
		Enriched: ch.Enriched,
	}

}

func (c *Cache) loadSportsMatches() ([]SportsMatchDTO, error) {

	matches, err := c.client.Matches()

	if err != nil {

		return nil, err

	}

	out := make([]SportsMatchDTO, 0, len(matches))

	for _, m := range matches {

		out = append(out, sportsMatchFromMatch(m))

	}

	return out, nil

}

func sportsMatchFromMatch(m mediakit.Match) SportsMatchDTO {

	dto := SportsMatchDTO{

		ID: m.ID,
		Title: m.Title,
		Category: m.Category,
		League: m.League,

		HomeScore: m.HomeScore,
		AwayScore: m.AwayScore,
		StatusDetail: m.StatusDetail,
		Status: m.Status,
		Delayed: m.Delayed,

		StartsAt: m.StartTime.Unix(),
		Live: m.Live,

		Broadcast: m.Broadcast,
		Broadcasts: append([]string(nil), m.Broadcasts...),
	}

	if m.HomeTeam != nil {

		dto.HomeTeam = m.HomeTeam.Name
		dto.HomeShortName = m.HomeTeam.ShortName
		dto.HomeLogo = m.HomeTeam.Logo

	}

	if m.AwayTeam != nil {

		dto.AwayTeam = m.AwayTeam.Name
		dto.AwayShortName = m.AwayTeam.ShortName
		dto.AwayLogo = m.AwayTeam.Logo

	}

	if m.Channel != nil {

		dto.Channel = &MatchedChannelDTO{

			ID: m.Channel.ChannelID,
			Name: m.Channel.Name,
			Logo: m.Channel.Logo,
		}

	}

	return dto

}

// popularChannelNames is a curated ranking of well-known channels, checked
// case-insensitively against the (possibly enrichment-renamed) channel name.
// Channels not in this list keep their existing relative order after it.
var popularChannelNames = []string{

	"ESPN", "ESPN2", "ESPNU", "ESPN News",
	"ABC", "CBS", "NBC", "FOX", "The CW",
	"CNN", "Fox News", "MSNBC", "CNBC",
	"HBO", "Showtime", "Starz",
	"TNT", "TBS", "USA Network", "FX", "FXX", "AMC", "Syfy",
	"Discovery Channel", "History", "National Geographic", "TLC", "A&E", "Lifetime",
	"Comedy Central", "MTV", "VH1", "BET",
	"Disney Channel", "Nickelodeon", "Cartoon Network",
	"Fox Sports 1", "Fox Sports 2", "NBA TV", "NFL Network", "MLB Network", "NHL Network", "Golf Channel",
	"Bravo", "E!", "Food Network", "HGTV", "Travel Channel", "Weather Channel",
}

func rankPopularChannels(channels []LiveChannelDTO) []LiveChannelDTO {

	rank := make(map[string]int, len(popularChannelNames))

	for i, name := range popularChannelNames {

		rank[strings.ToLower(name)] = i

	}

	ranked := dedupeChannelsByName(channels)

	sort.SliceStable(ranked, func(i, j int) bool {

		// Prefer channels with verified metadata + logos.
		hi, hj := ranked[i].Enriched && ranked[i].Logo != "", ranked[j].Enriched && ranked[j].Logo != ""

		if hi != hj {

			return hi

		}

		ri, oki := rank[strings.ToLower(ranked[i].Name)]
		rj, okj := rank[strings.ToLower(ranked[j].Name)]

		if oki && okj {

			return ri < rj

		}

		return oki && !okj

	})

	return ranked

}

// dedupeChannelsByName collapses channels that share a name (e.g. multiple
// regional "ESPN" entries) down to one, preferring the enriched/logo-bearing
// variant, and keeps the first-seen position for ordering.
func dedupeChannelsByName(channels []LiveChannelDTO) []LiveChannelDTO {

	bestByName := make(map[string]LiveChannelDTO, len(channels))
	order := make([]string, 0, len(channels))

	for _, ch := range channels {

		key := strings.ToLower(strings.TrimSpace(ch.Name))

		if key == "" {

			continue

		}

		existing, ok := bestByName[key]

		if !ok {

			bestByName[key] = ch
			order = append(order, key)

			continue

		}

		if channelIconPriority(ch) > channelIconPriority(existing) {

			bestByName[key] = ch

		}

	}

	out := make([]LiveChannelDTO, 0, len(order))

	for _, key := range order {

		out = append(out, bestByName[key])

	}

	return out

}

func channelIconPriority(ch LiveChannelDTO) int {

	score := 0

	if ch.Enriched {

		score += 2

	}

	if ch.Logo != "" {

		score++

	}

	return score

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
