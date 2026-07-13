package discover

import "time"

// FeedItem is a home-feed title. TMDB metadata is always present; Showbox id
// is filled when already cached, otherwise resolved on click.
type FeedItem struct {

	ID int `json:"id"`
	TMDBID int `json:"tmdbId"`
	Kind string `json:"kind"`

	Title string `json:"title"`
	Year int `json:"year"`

	Poster string `json:"poster"`
	Backdrop string `json:"backdrop,omitempty"`
	Description string `json:"description"`
	Rating string `json:"rating"`

	Genres []string `json:"genres,omitempty"`
	Runtime int `json:"runtime,omitempty"`
	MatchReason string `json:"matchReason,omitempty"`

}

// FeedSection is one horizontal row on the home feed.
type FeedSection struct {

	ID string `json:"id"`
	Title string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`

	Kind string `json:"kind"`
	Items []FeedItem `json:"items"`

}

// HomeFeed is the single-payload home response for movies or shows.
type HomeFeed struct {

	Featured []FeedItem `json:"featured,omitempty"`
	Sections []FeedSection `json:"sections"`

	RefreshedAt time.Time `json:"refreshedAt"`

}

// ResolveResult is a Showbox-mapped title ready for detail/watch navigation.
type ResolveResult struct {

	ID int `json:"id"`
	TMDBID int `json:"tmdbId"`
	Kind string `json:"kind"`

	Title string `json:"title"`
	Year int `json:"year"`
	Poster string `json:"poster"`

}
