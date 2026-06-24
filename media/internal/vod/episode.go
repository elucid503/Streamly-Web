package vod

import (
	"fmt"
	"time"

	"mediakit/internal/febbox"
	"mediakit/internal/fileparser"
	"mediakit/internal/intro"
	"mediakit/internal/introdb"
	"mediakit/internal/meta"
	"mediakit/internal/quality"
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

// Info returns episode metadata from IMDb, falling back to Showbox fields.
func (e *Episode) Info() (EpisodeInfo, error) {

	details, err := e.show.Details()

	if err != nil {

		return EpisodeInfo{}, err

	}

	info := EpisodeInfo{}

	if details.IMDBId != "" {

		if epMeta, ok := e.show.deps.GetEpisodeMeta(details.IMDBId, e.season, e.episode); ok {

			info = epMeta

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

func (e *Episode) titleFallback(details meta.TitleDetails) (string, error) {

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

		ID:   file.FID,
		Name: file.FileName,

		Season:  e.season,
		Episode: e.episode,

		shareKey: shareKey,
	}, nil

}

// Qualities lists available download renditions for this episode.
// Share-key folders are tried first, then Vixsrc (TMDB).
func (e *Episode) Qualities() ([]quality.Quality, error) {

	details, err := e.show.Details()

	if err != nil {

		streamDebugf("show %d S%02dE%02d details error: %v", e.show.id, e.season, e.episode, err)

	} else {

		streamDebugf("show %d S%02dE%02d metadata tmdb=%d imdb=%q", e.show.id, e.season, e.episode, details.TMDBId, details.IMDBId)

	}

	streamDebugf("show %d S%02dE%02d trying share-key path", e.show.id, e.season, e.episode)

	if qualities, ok := e.tryShareKeyQualities(); ok {

		streamDebugf("show %d S%02dE%02d share-key path ok count=%d", e.show.id, e.season, e.episode, len(qualities))

		return qualities, nil

	}

	if err == nil && details.TMDBId > 0 {

		if qualities, ok := providerQualities(e.show.deps, details.TMDBId, "tv", e.season, e.episode); ok {

			return qualities, nil

		}

		streamDebugf("show %d S%02dE%02d vixsrc path miss", e.show.id, e.season, e.episode)

	}

	return []quality.Quality{}, nil

}

func (e *Episode) tryShareKeyQualities() ([]quality.Quality, bool) {

	qualities, err := e.shareKeyQualities()

	if err != nil {

		streamDebugf("show %d S%02dE%02d share-key path error: %v", e.show.id, e.season, e.episode, err)

		return nil, false

	}

	if len(qualities) == 0 {

		streamDebugf("show %d S%02dE%02d share-key path miss", e.show.id, e.season, e.episode)

		return nil, false

	}

	return qualities, true

}

func (e *Episode) consoleQualities(imdbID string) ([]quality.Quality, bool) {

	fid, err := e.show.deps.GetConsoleEpisodeFID(imdbID, e.season, e.episode)

	if err != nil || fid <= 0 {

		return nil, false

	}

	items, err := e.show.deps.GetConsoleLinks(fid)

	if err != nil || len(items) == 0 {

		return nil, false

	}

	return quality.ToQualities(items), true

}

func (e *Episode) shareKeyQualities() ([]quality.Quality, error) {

	shareKey, err := e.show.ShareKey()

	if err != nil {

		return nil, err

	}

	files, err := e.allEpisodeFiles(shareKey)

	if err != nil {

		return nil, err

	}

	if len(files) == 0 {

		return []quality.Quality{}, nil

	}

	file := fileparser.BestSourceFile(files)

	items, err := e.show.deps.GetLinks(shareKey, file.FID, "")

	if err != nil {

		return nil, err

	}

	qualities := quality.ToQualities(items)

	if source1080, ok := fileparser.BestSourceFileAtHeight(files, 1080); ok {

		if originalURL, err := e.show.deps.GetDownloadURL(shareKey, source1080.FID, ""); err == nil {

			qualities = quality.WithOriginalAtHeight(qualities, originalURL, source1080.FileName, 1080)

		}

	}

	if quality.NeedsOriginalFallback(qualities) {

		if originalURL, err := e.show.deps.GetDownloadURL(shareKey, file.FID, ""); err == nil {

			qualities = quality.WithOriginalFallback(qualities, originalURL, file.FileName)

		}

	}

	return qualities, nil

}

func (e *Episode) allEpisodeFiles(shareKey string) ([]febbox.File, error) {

	season := e.show.Season(e.season)

	folder, err := season.resolveFolder(shareKey)

	if err == nil {

		children, listErr := e.show.listFiles(shareKey, folder.FID)

		if listErr != nil {

			return nil, listErr

		}

		return fileparser.AllEpisodeFiles(fileparser.FilesOnly(children), e.season, e.episode), nil

	}

	root, listErr := e.show.listFiles(shareKey, 0)

	if listErr != nil {

		return nil, listErr

	}

	if direct := fileparser.FilesOnly(root); len(direct) > 0 {

		return fileparser.AllEpisodeFiles(direct, e.season, e.episode), nil

	}

	var matches []febbox.File

	for _, item := range fileparser.ParseSeasons(root) {

		children, listErr := e.show.listFiles(shareKey, item.Folder.FID)

		if listErr != nil {

			continue

		}

		matches = append(matches, fileparser.AllEpisodeFiles(fileparser.FilesOnly(children), e.season, e.episode)...)

	}

	return matches, nil

}

// BestQuality picks the rendition closest to targetHeight pixels.
func (e *Episode) BestQuality(targetHeight int) (*quality.Quality, error) {

	qualities, err := e.Qualities()

	if err != nil {

		return nil, err

	}

	picked := quality.PickQuality(qualities, targetHeight)

	if picked == nil {

		return nil, fmt.Errorf("episode S%02dE%02d: no qualities available", e.season, e.episode)

	}

	return picked, nil

}

// StreamURL returns the best progressive or HLS URL at the target resolution.
func (e *Episode) StreamURL(targetHeight int) (string, error) {

	q, err := e.BestQuality(targetHeight)

	if err != nil {

		return "", err

	}

	return q.URL, nil

}

// Intro fetches intro timing from TheIntroDB for this episode.
func (e *Episode) Intro(opts ...intro.Option) (*intro.Data, error) {

	details, err := e.show.Details()

	if err != nil {

		return nil, err

	}

	cfg := intro.ApplyOptions(opts)

	query, err := intro.BuildQuery(details.TMDBId, details.IMDBId, e.season, e.episode, intro.DurationMs(cfg))

	if err != nil {

		return nil, err

	}

	record, err := e.show.deps.GetIntro(query)

	if err != nil {

		return nil, introdb.MapGetMediaError(err)

	}

	return intro.FromRecord(record), nil

}

// SkipIntroFrom returns the seek target to skip the intro from the current position.
func (e *Episode) SkipIntroFrom(position time.Duration) (time.Duration, error) {

	data, err := e.Intro()

	if err != nil {

		return 0, err

	}

	return introdb.IntroSkipTarget(intro.ToRecord(data), position)

}

// CreditsStart estimates when credits begin for auto-next timing.
func (e *Episode) CreditsStart(duration time.Duration) (time.Duration, bool) {

	data, err := e.Intro()

	if err != nil {

		return 0, false

	}

	return introdb.CreditsStart(intro.ToRecord(data), duration.Milliseconds())

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
		root, listErr := e.show.listFiles(shareKey, 0)

		if listErr != nil {

			return nil, err

		}

		for _, item := range fileparser.ParseEpisodes(fileparser.FilesOnly(root), e.season) {

			if item.Number == e.episode {

				e.file = &item.File
				return e.file, nil

			}

		}

		return nil, fmt.Errorf("episode S%02dE%02d not found", e.season, e.episode)

	}

	children, err := e.show.listFiles(shareKey, folder.FID)

	if err != nil {

		return nil, err

	}

	for _, item := range fileparser.ParseEpisodes(fileparser.FilesOnly(children), e.season) {

		if item.Number == e.episode {

			e.file = &item.File
			return e.file, nil

		}

	}

	return nil, fmt.Errorf("episode S%02dE%02d not found", e.season, e.episode)

}
