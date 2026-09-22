package mediakit

import (
	"mediakit/internal/live/catalog"
	"mediakit/internal/live/guide"
	"mediakit/internal/live/source"
	livesports "mediakit/internal/live/sports"
)

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
