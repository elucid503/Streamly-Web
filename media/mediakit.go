// Package mediakit provides VOD search, streaming, and live TV for the Streamly platform.
package mediakit

import (
	"time"

	"mediakit/internal/client"
	"mediakit/internal/discover"
	"mediakit/internal/intro"
	"mediakit/internal/live/catalog"
	"mediakit/internal/live/guide"
	"mediakit/internal/live/source"
	livesports "mediakit/internal/live/sports"
	"mediakit/internal/meta"
	"mediakit/internal/quality"
	"mediakit/internal/tmdb"
	"mediakit/internal/vod"
)

// Client is the entry point for catalogue search, VOD browsing, and live TV.
type Client = client.Client

// Option configures a Client.
type Option = client.Option

// MediaKind distinguishes movies from TV series.
type MediaKind = meta.MediaKind

const (
	MediaMovie = meta.MediaMovie
	MediaShow  = meta.MediaShow
)

// SearchHit is a single catalogue result from Showbox search.
type SearchHit = meta.SearchHit

// TitleDetails is user-facing metadata for a movie or show.
type TitleDetails = meta.TitleDetails

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

// Channel is a metadata-only live TV channel (no stream URL).
type Channel = catalog.Channel

// ChannelCatalog is the full live TV channel index.
type ChannelCatalog = catalog.Catalog

// Country is channel country metadata.
type Country = catalog.Country

// GuideEntry pairs a channel with current/next program data.
type GuideEntry = guide.Entry

// Program is a guide airing slot.
type Program = guide.Program

// Match is a single sports fixture.
type Match = livesports.Match

// DecodeSportsChannelID validates and decodes an opaque synthetic sports
// channel identity.
func DecodeSportsChannelID(id string) (string, bool) {

	return livesports.DecodeChannelID(id)

}

// MatchTeam is one side of a sports match.
type MatchTeam = livesports.Team

// MatchedChannel is an optional catalog channel soft-linked to a match.
type MatchedChannel = livesports.MatchedChannel

// Stream is a playable reference from the source-provider layer.
type Stream = source.Stream

// SourceProvider resolves catalog channel IDs into streams.
type SourceProvider = source.Provider

// PublicSourceProvider is an anonymized live source option for clients.
type PublicSourceProvider = source.PublicProvider

// ErrNoProviders is returned when no live stream providers are registered.
var ErrNoProviders = source.ErrNoProviders

// ErrStreamUnavailable is returned when providers fail to resolve a stream.
var ErrStreamUnavailable = source.ErrUnavailable

// TopCategory is a curated Showbox ranking list.
type TopCategory = discover.TopCategory

// TMDBClient fetches TMDB discover/trending lists.
type TMDBClient = tmdb.Client

// TMDBItem is a TMDB list result.
type TMDBItem = tmdb.Item

// TMDBKind is movie or tv for TMDB endpoints.
type TMDBKind = tmdb.MediaKind

const (
	TMDBMovie = tmdb.KindMovie
	TMDBTV    = tmdb.KindTV
)

// TMDBGenreName returns a display name for a TMDB genre id.
func TMDBGenreName(id int) string {

	return tmdb.GenreName(id)

}

// TMDBGenreNames maps genre ids to display names.
func TMDBGenreNames(ids []int) []string {

	return tmdb.GenreNames(ids)

}

// New builds a Client with optional configuration.
func New(opts ...Option) *Client {

	return client.New(opts...)

}

// WithChildMode sets the Showbox child-mode flag.
func WithChildMode(mode string) Option {

	return client.WithChildMode(mode)

}

// WithFebboxCookie sets the Febbox `ui` auth cookie required for quality links.
func WithFebboxCookie(cookie string) Option {

	return client.WithFebboxCookie(cookie)

}

// WithIntroDBKey sets an optional TheIntroDB API key.
func WithIntroDBKey(key string) Option {

	return client.WithIntroDBKey(key)

}

// WithTMDBAPIKey sets the TMDB v3 API key for episode and title metadata.
func WithTMDBAPIKey(key string) Option {

	return client.WithTMDBAPIKey(key)

}

// WithTVBaseURL is retained for compatibility; live catalog no longer uses a provider origin.
func WithTVBaseURL(baseURL string) Option {

	return client.WithTVBaseURL(baseURL)

}

// WithIntroCache enables a 6-hour in-memory cache for TheIntroDB lookups.
func WithIntroCache(enabled bool) Option {

	return client.WithIntroCache(enabled)

}

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

// PickQuality chooses the best source at or below targetHeight, preferring progressive files.
func PickQuality(qualities []Quality, targetHeight int) *Quality {

	return quality.PickQuality(qualities, targetHeight)

}

// PickNextLowerQuality returns the next lower rendition below belowHeight.
func PickNextLowerQuality(qualities []Quality, belowHeight int) *Quality {

	return quality.PickNextLowerQuality(qualities, belowHeight)

}
