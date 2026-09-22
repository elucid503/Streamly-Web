package mediakit

import (
	"time"

	"mediakit/internal/playback/quality"
	"mediakit/internal/vod"
	"mediakit/internal/vod/intro"
)

// Movie is a chainable handle for a film.
type Movie = vod.Movie

// Show is a chainable handle for a TV series.
type Show = vod.Show

// Season is a chainable handle for one season of a TV show.
type Season = vod.Season

// Episode is a chainable handle for one episode of a TV show.
type Episode = vod.Episode

// MediaFile is a playable file inside a Febbox share.
type MediaFile = vod.MediaFile

// EpisodeInfo is display metadata for one episode.
type EpisodeInfo = vod.EpisodeInfo

// ShowSeasonInfo is a summary of one TV season from TMDB.
type ShowSeasonInfo = vod.ShowSeasonInfo

// Quality is one downloadable rendition of a video file.
type Quality = quality.Quality

// IntroData is normalized intro timing from TheIntroDB.
type IntroData = intro.Data

// IntroSegment is a community-verified intro/recap/credits window.
type IntroSegment = intro.Segment

// IntroOption configures TheIntroDB lookups.
type IntroOption = intro.Option

// WithDuration hints the media runtime for duration-aware intro matching.
func WithDuration(d time.Duration) IntroOption {

	return intro.WithDuration(d)

}

// IsHLSURL reports whether a URL points at an HLS playlist.
func IsHLSURL(raw string) bool {

	return quality.IsHLSURL(raw)

}

// IsWebPlayableURL reports whether a URL points at a browser-friendly container.
func IsWebPlayableURL(raw string) bool {

	return quality.IsWebPlayableURL(raw)

}
