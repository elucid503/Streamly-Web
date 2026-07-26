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

// LiveChannelDTO is a live TV channel entry (metadata only; no stream URL).
type LiveChannelDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Code string `json:"code"`
	Logo string `json:"logo"`

	Country string `json:"country"`
	CountryName string `json:"countryName,omitempty"`
	Category string `json:"category"`
	Categories []string `json:"categories,omitempty"`
	Network string `json:"network,omitempty"`
	Owners []string `json:"owners,omitempty"`
	Website string `json:"website,omitempty"`
	Enriched bool `json:"enriched"`
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
	League string `json:"league,omitempty"`

	HomeTeam string `json:"homeTeam,omitempty"`
	AwayTeam string `json:"awayTeam,omitempty"`
	HomeLogo string `json:"homeLogo,omitempty"`
	AwayLogo string `json:"awayLogo,omitempty"`

	HomeScore    *int   `json:"homeScore,omitempty"`
	AwayScore    *int   `json:"awayScore,omitempty"`
	StatusDetail string `json:"statusDetail,omitempty"`
	// Status is scoreboard lifecycle when known: pre / in / post.
	Status string `json:"status,omitempty"`

	StartsAt int64 `json:"startsAt"`
	Live     bool  `json:"live"`

	// Broadcast is the primary live TV/stream outlet from the scoreboard (e.g. "SNY").
	Broadcast string `json:"broadcast,omitempty"`
	// Broadcasts lists all known outlets for this event.
	Broadcasts []string `json:"broadcasts,omitempty"`

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
