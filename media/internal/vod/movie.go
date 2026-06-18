package vod

import (
	"fmt"
	"sync"
	"time"

	"mediakit/internal/febbox"
	"mediakit/internal/fileparser"
	"mediakit/internal/intro"
	"mediakit/internal/introdb"
	"mediakit/internal/meta"
	"mediakit/internal/quality"
	"mediakit/internal/subtitle"
)

// Movie is a chainable handle for a film.
type Movie struct {

	deps Deps
	id int

	mu sync.Mutex
	details *meta.TitleDetails

	shareKey string
	shareErr error
	shareSet bool

	file *febbox.File

}

// NewMovie creates a Movie handle for the given Showbox id.
func NewMovie(deps Deps, id int) *Movie {

	return &Movie{deps: deps, id: id}

}

// ID returns the Showbox catalogue id.
func (m *Movie) ID() int {

	return m.id

}

// Details fetches and caches movie metadata.
func (m *Movie) Details() (meta.TitleDetails, error) {

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.details != nil {

		return *m.details, nil

	}

	details, err := m.deps.GetMovieDetails(m.id)

	if err != nil {

		return meta.TitleDetails{}, err

	}

	m.details = &details

	return details, nil

}

// ShareKey resolves the Febbox share key that hosts this movie's files.
func (m *Movie) ShareKey() (string, error) {

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shareSet {

		return m.shareKey, m.shareErr

	}

	m.shareKey, m.shareErr = m.deps.GetFebBoxID(m.id, 1) // 1 = showbox.BoxMovie
	m.shareSet = true

	return m.shareKey, m.shareErr

}

// File resolves the primary playable file for this movie.
func (m *Movie) File() (*MediaFile, error) {

	file, err := m.resolveFile()

	if err != nil {

		return nil, err

	}

	if file == nil {

		return nil, fmt.Errorf("movie %d: no playable file found", m.id)

	}

	shareKey, err := m.ShareKey()

	if err != nil {

		return nil, err

	}

	return &MediaFile{

		ID: file.FID,
		Name: file.FileName,

		shareKey: shareKey,

	}, nil

}

// Qualities lists available download renditions for this movie.
func (m *Movie) Qualities() ([]quality.Quality, error) {

	shareKey, err := m.ShareKey()

	if err != nil {

		return nil, err

	}

	root, err := m.deps.ListFiles(shareKey, 0, "")

	if err != nil {

		return nil, err

	}

	direct := fileparser.FilesOnly(root)

	if len(direct) == 0 {

		seasons := fileparser.SeasonsOnly(root)

		if len(seasons) == 0 {

			return nil, fmt.Errorf("movie %d: no playable file found", m.id)

		}

		children, err := m.deps.ListFiles(shareKey, seasons[0].FID, "")

		if err != nil {

			return nil, err

		}

		direct = fileparser.FilesOnly(children)

	}

	if len(direct) == 0 {

		return nil, fmt.Errorf("movie %d: no playable file found", m.id)

	}

	file := fileparser.BestSourceFile(direct) // Pick the highest-resolution source file.
	items, err := m.deps.GetLinks(shareKey, file.FID, "")

	if err != nil {

		return nil, err

	}

	qualities := quality.ToQualities(items)

	if source1080, ok := fileparser.BestSourceFileAtHeight(direct, 1080); ok {

		if originalURL, err := m.deps.GetDownloadURL(shareKey, source1080.FID, ""); err == nil {

			qualities = quality.WithOriginalAtHeight(qualities, originalURL, source1080.FileName, 1080)

		}

	}

	if quality.NeedsOriginalFallback(qualities) {

		if originalURL, err := m.deps.GetDownloadURL(shareKey, file.FID, ""); err == nil {

			qualities = quality.WithOriginalFallback(qualities, originalURL, file.FileName)

		}

	}

	return qualities, nil

}

// Subtitles lists external subtitle files found alongside the movie video.
func (m *Movie) Subtitles() ([]subtitle.Subtitle, error) {

	shareKey, err := m.ShareKey()

	if err != nil {

		return nil, err

	}

	siblings, video, err := m.listSiblingFiles()

	if err != nil {

		return nil, err

	}

	return subtitle.CollectSubtitles(shareKey, siblings, video), nil

}

func (m *Movie) listSiblingFiles() ([]febbox.File, *febbox.File, error) {

	video, err := m.resolveFile()

	if err != nil {

		return nil, nil, err

	}

	if video == nil {

		return nil, nil, fmt.Errorf("movie %d: no playable file found", m.id)

	}

	shareKey, err := m.ShareKey()

	if err != nil {

		return nil, nil, err

	}

	root, err := m.deps.ListFiles(shareKey, 0, "")

	if err != nil {

		return nil, nil, err

	}

	direct := fileparser.FilesOnly(root)

	if len(direct) > 0 {

		return direct, video, nil

	}

	seasons := fileparser.SeasonsOnly(root)

	if len(seasons) == 0 {

		return direct, video, nil

	}

	children, err := m.deps.ListFiles(shareKey, seasons[0].FID, "")

	if err != nil {

		return nil, nil, err

	}

	return fileparser.FilesOnly(children), video, nil

}

// BestQuality picks the rendition closest to targetHeight pixels.
func (m *Movie) BestQuality(targetHeight int) (*quality.Quality, error) {

	qualities, err := m.Qualities()

	if err != nil {

		return nil, err

	}

	picked := quality.PickQuality(qualities, targetHeight)

	if picked == nil {

		return nil, fmt.Errorf("movie %d: no qualities available", m.id)

	}

	return picked, nil

}

// StreamURL returns the best progressive or HLS URL at the target resolution.
func (m *Movie) StreamURL(targetHeight int) (string, error) {

	q, err := m.BestQuality(targetHeight)

	if err != nil {

		return "", err

	}

	return q.URL, nil

}

// Intro fetches intro timing from TheIntroDB for this movie.
func (m *Movie) Intro(opts ...intro.Option) (*intro.Data, error) {

	details, err := m.Details()

	if err != nil {

		return nil, err

	}

	cfg := intro.ApplyOptions(opts)

	query, err := intro.BuildQuery(details.TMDBId, details.IMDBId, 0, 0, intro.DurationMs(cfg))

	if err != nil {

		return nil, err

	}

	record, err := m.deps.GetIntro(query)

	if err != nil {

		return nil, introdb.MapGetMediaError(err)

	}

	return intro.FromRecord(record), nil

}

// SkipIntroFrom returns the seek target to skip the intro from the current position.
func (m *Movie) SkipIntroFrom(position time.Duration) (time.Duration, error) {

	data, err := m.Intro()

	if err != nil {

		return 0, err

	}

	return introdb.IntroSkipTarget(intro.ToRecord(data), position)

}

// CreditsStart estimates when credits begin.
func (m *Movie) CreditsStart(duration time.Duration) (time.Duration, bool) {

	data, err := m.Intro()

	if err != nil {

		return 0, false

	}

	return introdb.CreditsStart(intro.ToRecord(data), duration.Milliseconds())

}

func (m *Movie) resolveFile() (*febbox.File, error) {

	if m.file != nil {

		return m.file, nil

	}

	shareKey, err := m.ShareKey()

	if err != nil {

		return nil, err

	}

	if shareKey == "" {

		return nil, fmt.Errorf("movie %d: no febbox share key", m.id)

	}

	root, err := m.deps.ListFiles(shareKey, 0, "")

	if err != nil {

		return nil, err

	}

	direct := fileparser.FilesOnly(root)

	if len(direct) > 0 {

		m.file = &direct[0]
		return m.file, nil

	}

	seasons := fileparser.SeasonsOnly(root)

	if len(seasons) == 0 {

		return nil, nil

	}

	children, err := m.deps.ListFiles(shareKey, seasons[0].FID, "")

	if err != nil {

		return nil, err

	}

	files := fileparser.FilesOnly(children)

	if len(files) == 0 {

		return nil, nil

	}

	m.file = &files[0]

	return m.file, nil

}
