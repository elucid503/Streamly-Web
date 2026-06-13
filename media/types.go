package mediakit

import "time"

// MediaKind distinguishes movies from TV series.
type MediaKind int

const (

	MediaMovie MediaKind = 1
	MediaShow MediaKind = 2

)

// SearchHit is a single catalogue result from Showbox search.
type SearchHit struct {

	ID int
	Kind MediaKind

	Title string
	Year int
	Poster string
	Description string

	IMDBRating string

}

// TitleDetails is user-facing metadata for a movie or show.
type TitleDetails struct {

	Title string
	Year string

	Poster string
	Banner string
	Description string

	IMDBRating string

	TMDBId int
	IMDBId string

	EpisodeTitles map[string]string

}

// MediaFile is a playable file inside a Febbox share.
type MediaFile struct {

	ID int

	Name string

	Season int
	Episode int

	shareKey string

}

// Quality is one downloadable rendition of a video file.
type Quality struct {

	URL string

	Label string
	Name string

	Speed string

	Size string
	Height int

	IsHLS bool

}

// IntroSegment is a community-verified intro/recap/credits window.
type IntroSegment struct {

	Start time.Duration
	End *time.Duration

}

// IntroData is normalized intro timing from TheIntroDB.
type IntroData struct {

	TMDBId int
	Type string
	Segments []IntroSegment

}

// LiveChannelInfo describes a live TV channel from the catalog.
type LiveChannelInfo struct {

	ID string
	DaddyID string

	Name string
	Slug string
	Logo string

	Country string

	Category string

	Status string

}

// LiveStream is a resolved live TV HLS playlist.
type LiveStream struct {

	URL string
	Referer string

	Channel LiveChannelInfo

}
