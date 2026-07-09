package catalog

import "time"

// SearchResultDTO is a single catalogue entry used in search results and trending lists.
type SearchResultDTO struct {
	ID   int    `json:"id"`
	Kind string `json:"kind"`

	Title string `json:"title"`
	Year  int    `json:"year"`

	Poster      string `json:"poster"`
	Description string `json:"description"`
	Rating      string `json:"rating"`
}

// CategoryDTO is a curated browse category.
type CategoryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// LiveChannelDTO is a live TV channel entry.
type LiveChannelDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Code string `json:"code"`
	Logo string `json:"logo"`

	Country  string `json:"country"`
	Category string `json:"category"`
	Enriched bool   `json:"enriched"`
}

// MatchedChannelDTO is the 24/7 channel a sports match's broadcast was matched to.
type MatchedChannelDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

// SportsMatchDTO is a live or upcoming sports fixture.
type SportsMatchDTO struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`

	HomeTeam string `json:"homeTeam,omitempty"`
	AwayTeam string `json:"awayTeam,omitempty"`

	HomeScore    *int   `json:"homeScore,omitempty"`
	AwayScore    *int   `json:"awayScore,omitempty"`
	StatusDetail string `json:"statusDetail,omitempty"`
	// Status is scoreboard lifecycle when known: pre / in / post.
	Status string `json:"status,omitempty"`

	StartsAt int64 `json:"startsAt"`
	Live     bool  `json:"live"`

	Channel *MatchedChannelDTO `json:"channel,omitempty"`
}

// Snapshot is an immutable point-in-time view of the catalog cache.
type Snapshot struct {
	movieTrending []SearchResultDTO
	showTrending  []SearchResultDTO

	movieCategories []CategoryDTO
	showCategories  []CategoryDTO

	movieCategoryTitles map[string][]SearchResultDTO
	showCategoryTitles  map[string][]SearchResultDTO

	liveChannels []LiveChannelDTO
	livePopular  []LiveChannelDTO

	sportsMatches []SportsMatchDTO

	searchIndex []SearchResultDTO
	refreshedAt time.Time
}

// SearchIndex returns the full-text search index built from the catalog.
func (s Snapshot) SearchIndex() []SearchResultDTO {

	return s.searchIndex

}
