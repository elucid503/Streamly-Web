package catalog

import "time"

// SearchResultDTO is a single catalogue entry used in search results and trending lists.
type SearchResultDTO struct {

	ID int `json:"id"`
	Kind string `json:"kind"`

	Title string `json:"title"`
	Year int `json:"year"`

	Poster string `json:"poster"`
	Description string `json:"description"`
	Rating string `json:"rating"`

}

// CategoryDTO is a curated browse category.
type CategoryDTO struct {

	ID string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`

}

// LiveChannelDTO is a live TV channel entry.
type LiveChannelDTO struct {

	ID string `json:"id"`
	DaddyID string `json:"daddyId"`

	Name string `json:"name"`
	Slug string `json:"slug"`
	Logo string `json:"logo"`

	Country string `json:"country"`
	Category string `json:"category"`

}

// Snapshot is an immutable point-in-time view of the catalog cache.
type Snapshot struct {

	movieTrending []SearchResultDTO
	showTrending []SearchResultDTO

	movieCategories []CategoryDTO
	showCategories []CategoryDTO

	movieCategoryTitles map[string][]SearchResultDTO
	showCategoryTitles map[string][]SearchResultDTO

	liveChannels []LiveChannelDTO
	livePopular []LiveChannelDTO

	searchIndex []SearchResultDTO
	refreshedAt time.Time

}

// SearchIndex returns the full-text search index built from the catalog.
func (s Snapshot) SearchIndex() []SearchResultDTO {

	return s.searchIndex

}
