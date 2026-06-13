package mediakit

import (
	"fmt"
	"time"

	"mediakit/internal/febbox"
	"mediakit/internal/introdb"
)

// Episode is a chainable handle for one episode of a TV show.
type Episode struct {

	show *Show

	season int
	episode int

	file *febbox.File

}

// SeasonNumber returns the season number.
func (e *Episode) SeasonNumber() int {

	return e.season

}

// Number returns the episode number.
func (e *Episode) Number() int {

	return e.episode

}

// EpisodeInfo is display metadata for one episode.
type EpisodeInfo struct {

	Title string
	Description string

	Poster string

}

// Info returns episode metadata from IMDb, falling back to Showbox fields.
func (e *Episode) Info() (EpisodeInfo, error) {

	details, err := e.show.Details()

	if err != nil {

		return EpisodeInfo{}, err

	}

	info := EpisodeInfo{}

	if details.IMDBId != "" {

		if meta, ok := e.show.client.imdb.Episode(details.IMDBId, e.season, e.episode); ok {

			info = EpisodeInfo{

				Title: meta.Title,
				Description: meta.Description,

				Poster: meta.Poster,

			}

		}

	}

	if info.Title == "" {

		info.Title, _ = e.titleFallback(details)

	}

	if info.Poster == "" && details.Poster != "" {

		info.Poster = details.Poster

	}

	if info.Description == "" && details.Description != "" {

		info.Description = details.Description

	}

	return info, nil

}

// Title returns the episode title from IMDb, falling back to Showbox metadata.
func (e *Episode) Title() (string, error) {

	info, err := e.Info()

	if err != nil {

		return "", err

	}

	if info.Title != "" {

		return info.Title, nil

	}

	return fmt.Sprintf("S%02dE%02d", e.season, e.episode), nil

}

func (e *Episode) titleFallback(details TitleDetails) (string, error) {

	if details.EpisodeTitles != nil {

		key := fmt.Sprintf("%d:%d", e.season, e.episode)

		if title, ok := details.EpisodeTitles[key]; ok && title != "" {

			return title, nil

		}

	}

	return fmt.Sprintf("S%02dE%02d", e.season, e.episode), nil

}

// File resolves the Febbox file entry for this episode.
func (e *Episode) File() (*MediaFile, error) {

	file, err := e.resolveFile()

	if err != nil {

		return nil, err

	}

	if file == nil {

		return nil, fmt.Errorf("episode S%02dE%02d not found", e.season, e.episode)

	}

	shareKey, err := e.show.ShareKey()

	if err != nil {

		return nil, err

	}

	return &MediaFile{

		ID: file.FID,
		Name: file.FileName,

		Season: e.season,
		Episode: e.episode,

		shareKey: shareKey,

	}, nil

}

// Qualities lists available download renditions for this episode.
func (e *Episode) Qualities() ([]Quality, error) {

	file, err := e.resolveFile()

	if err != nil {

		return nil, err

	}

	if file == nil {

		return nil, fmt.Errorf("episode S%02dE%02d not found", e.season, e.episode)

	}

	shareKey, err := e.show.ShareKey()

	if err != nil {

		return nil, err

	}

	items, err := e.show.client.febbox.GetLinks(shareKey, file.FID, "")

	if err != nil {

		return nil, err

	}

	return toQualities(items), nil

}

// Subtitles lists external subtitle files found alongside the episode video.
func (e *Episode) Subtitles() ([]Subtitle, error) {

	shareKey, err := e.show.ShareKey()

	if err != nil {

		return nil, err

	}

	siblings, video, err := e.listSiblingFiles()

	if err != nil {

		return nil, err

	}

	return collectSubtitles(shareKey, siblings, video), nil

}

func (e *Episode) listSiblingFiles() ([]febbox.File, *febbox.File, error) {

	video, err := e.resolveFile()

	if err != nil {

		return nil, nil, err

	}

	shareKey, err := e.show.ShareKey()

	if err != nil {

		return nil, nil, err

	}

	season := e.show.Season(e.season)

	folder, err := season.resolveFolder(shareKey)

	if err != nil {

		root, listErr := e.show.client.febbox.ListFiles(shareKey, 0, "")

		if listErr != nil {

			return nil, nil, err

		}

		return filesOnly(root), video, nil

	}

	children, err := e.show.client.febbox.ListFiles(shareKey, folder.FID, "")

	if err != nil {

		return nil, nil, err

	}

	return filesOnly(children), video, nil

}

// BestQuality picks the rendition closest to targetHeight pixels.
func (e *Episode) BestQuality(targetHeight int) (*Quality, error) {

	qualities, err := e.Qualities()

	if err != nil {

		return nil, err

	}

	picked := PickQuality(qualities, targetHeight)

	if picked == nil {

		return nil, fmt.Errorf("episode S%02dE%02d: no qualities available", e.season, e.episode)

	}

	return picked, nil

}

// StreamURL returns the best progressive or HLS URL at the target resolution.
func (e *Episode) StreamURL(targetHeight int) (string, error) {

	quality, err := e.BestQuality(targetHeight)

	if err != nil {

		return "", err

	}

	return quality.URL, nil

}

// Intro fetches intro timing from TheIntroDB for this episode.
func (e *Episode) Intro(opts ...IntroOption) (*IntroData, error) {

	details, err := e.show.Details()

	if err != nil {

		return nil, err

	}

	cfg := introConfig{}

	for _, opt := range opts {

		opt(&cfg)

	}

	query, err := introdb.QueryForTitle(details.TMDBId, details.IMDBId, e.season, e.episode, cfg.durationMs)

	if err != nil {

		return nil, err

	}

	record, err := e.show.client.intro.GetMedia(query)

	if err != nil {

		return nil, introdb.MapGetMediaError(err)

	}

	return toIntroData(record), nil

}

// SkipIntroFrom returns the seek target to skip the intro from the current position.
func (e *Episode) SkipIntroFrom(position time.Duration) (time.Duration, error) {

	data, err := e.Intro()

	if err != nil {

		return 0, err

	}

	record := toMediaRecord(data)

	return introdb.IntroSkipTarget(record, position)

}

// CreditsStart estimates when credits begin for auto-next timing.
func (e *Episode) CreditsStart(duration time.Duration) (time.Duration, bool) {

	data, err := e.Intro()

	if err != nil {

		return 0, false

	}

	record := toMediaRecord(data)

	return introdb.CreditsStart(record, duration.Milliseconds())

}

func (e *Episode) resolveFile() (*febbox.File, error) {

	if e.file != nil {

		return e.file, nil

	}

	shareKey, err := e.show.ShareKey()

	if err != nil {

		return nil, err

	}

	season := e.show.Season(e.season)

	folder, err := season.resolveFolder(shareKey)

	if err != nil {

		// Flat listing fallback when no season folders exist.
		root, listErr := e.show.client.febbox.ListFiles(shareKey, 0, "")

		if listErr != nil {

			return nil, err

		}

		for _, item := range parseEpisodes(filesOnly(root), e.season) {

			if item.Number == e.episode {

				e.file = &item.File
				return e.file, nil

			}

		}

		return nil, fmt.Errorf("episode S%02dE%02d not found", e.season, e.episode)

	}

	children, err := e.show.client.febbox.ListFiles(shareKey, folder.FID, "")

	if err != nil {

		return nil, err

	}

	for _, item := range parseEpisodes(filesOnly(children), e.season) {

		if item.Number == e.episode {

			e.file = &item.File
			return e.file, nil

		}

	}

	return nil, fmt.Errorf("episode S%02dE%02d not found", e.season, e.episode)

}
