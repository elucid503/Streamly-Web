package catalog

import (
	"time"

	catalogindex "streamly/internal/features/catalog/index"
	"streamly/internal/features/catalog/vod"
)

type SearchResultDTO = catalogindex.SearchResultDTO

type CategoryDTO = catalogindex.CategoryDTO

type LiveChannelDTO = catalogindex.LiveChannelDTO

type SportsMatchDTO = catalogindex.SportsMatchDTO

type MatchedChannelDTO = catalogindex.MatchedChannelDTO

type SeasonDTO = vod.SeasonDTO

type EpisodeDTO = vod.EpisodeDTO

// TitleDetailsDTO is user-facing metadata for a movie or show.
type TitleDetailsDTO struct {

	ID int `json:"id"`
	Kind string `json:"kind"`

	Title string `json:"title"`
	Year string `json:"year"`

	Poster string `json:"poster"`
	Banner string `json:"banner,omitempty"`

	Description string `json:"description"`
	Rating string `json:"rating"`

}

type titleDetailsCacheEntry struct {

	details *TitleDetailsDTO
	fetchedAt time.Time

}

// QualityDTO is one downloadable rendition of a video file.
type QualityDTO struct {

	Label string `json:"label"`
	Height int `json:"height"`

	IsHLS bool `json:"isHls"`

	URL string `json:"url"`
	ProxyURL string `json:"proxyUrl,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

}

// StreamDTO is the resolved streaming payload returned to clients.
type StreamDTO struct {

	Qualities []QualityDTO `json:"qualities"`

}

// SubtitleDTO describes an external subtitle track.
type SubtitleDTO struct {

	ID string `json:"id"`
	Label string `json:"label"`

	Language string `json:"language"`
	Format string `json:"format"`

	ProxyURL string `json:"proxyUrl"`
	Source string `json:"source,omitempty"`

}

// IntroDTO carries skip-intro timing offsets.
type IntroDTO struct {

	IntroStartMs *int64 `json:"introStartMs,omitempty"`
	IntroEndMs *int64 `json:"introEndMs,omitempty"`

	CreditsStartMs *int64 `json:"creditsStartMs,omitempty"`

}

// NextEpisodeDTO identifies the episode after the current one.
type NextEpisodeDTO struct {

	Season int `json:"season"`
	Episode int `json:"episode"`

	Title string `json:"title"`

}

// LiveStreamRef is a resolved live TV stream (URL + optional playback headers).
type LiveStreamRef struct {

	URL string
	IsHLS bool
	Headers map[string]string
	// Provider is an anonymized public key (auto/s1/s2/…), never an upstream brand.
	Provider string

}

// LiveSourceOption is an anonymized provider choice for the player UI.
type LiveSourceOption struct {

	Key string `json:"key"`
	Label string `json:"label"`
	Description string `json:"description,omitempty"`

}
