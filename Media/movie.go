package mediakit

import (
	"fmt"
	"sync"
	"time"

	"mediakit/internal/febbox"
	"mediakit/internal/introdb"
	"mediakit/internal/showbox"
)

// Movie is a chainable handle for a film.
type Movie struct {
	client *Client
	id     int

	mu       sync.Mutex
	details  *TitleDetails
	shareKey string
	shareErr error
	shareSet bool
	file     *febbox.File
}

// ID returns the Showbox catalogue id.
func (m *Movie) ID() int {
	return m.id
}

// Details fetches and caches movie metadata.
func (m *Movie) Details() (TitleDetails, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.details != nil {
		return *m.details, nil
	}

	raw, err := m.client.showbox.GetMovie(m.id)
	if err != nil {
		return TitleDetails{}, err
	}

	details := parseTitleDetails(raw)
	if details.IMDBId != "" {
		if meta, err := m.client.imdb.Movie(details.IMDBId); err == nil {
			enrichTitleDetails(&details, meta)
		}
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

	m.shareKey, m.shareErr = m.client.showbox.GetFebBoxID(m.id, showbox.BoxMovie)
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
		ID:       file.FID,
		Name:     file.FileName,
		shareKey: shareKey,
	}, nil
}

// Qualities lists available download renditions for this movie.
func (m *Movie) Qualities() ([]Quality, error) {
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

	items, err := m.client.febbox.GetLinks(shareKey, file.FID, "")
	if err != nil {
		return nil, err
	}

	return toQualities(items), nil
}

// Subtitles lists external subtitle files found alongside the movie video.
func (m *Movie) Subtitles() ([]Subtitle, error) {
	shareKey, err := m.ShareKey()
	if err != nil {
		return nil, err
	}

	siblings, video, err := m.listSiblingFiles()
	if err != nil {
		return nil, err
	}
	return collectSubtitles(shareKey, siblings, video), nil
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

	root, err := m.client.febbox.ListFiles(shareKey, 0, "")
	if err != nil {
		return nil, nil, err
	}

	direct := filesOnly(root)
	if len(direct) > 0 {
		return direct, video, nil
	}

	seasons := seasonsOnly(root)
	if len(seasons) == 0 {
		return direct, video, nil
	}

	children, err := m.client.febbox.ListFiles(shareKey, seasons[0].FID, "")
	if err != nil {
		return nil, nil, err
	}
	return filesOnly(children), video, nil
}

// BestQuality picks the rendition closest to targetHeight pixels.
func (m *Movie) BestQuality(targetHeight int) (*Quality, error) {
	qualities, err := m.Qualities()
	if err != nil {
		return nil, err
	}

	picked := PickQuality(qualities, targetHeight)
	if picked == nil {
		return nil, fmt.Errorf("movie %d: no qualities available", m.id)
	}

	return picked, nil
}

// StreamURL returns the best progressive or HLS URL at the target resolution.
func (m *Movie) StreamURL(targetHeight int) (string, error) {
	quality, err := m.BestQuality(targetHeight)
	if err != nil {
		return "", err
	}
	return quality.URL, nil
}

// Intro fetches intro timing from TheIntroDB for this movie.
func (m *Movie) Intro(opts ...IntroOption) (*IntroData, error) {
	details, err := m.Details()
	if err != nil {
		return nil, err
	}

	cfg := introConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	query, err := introdb.QueryForTitle(details.TMDBId, details.IMDBId, 0, 0, cfg.durationMs)
	if err != nil {
		return nil, err
	}

	record, err := m.client.intro.GetMedia(query)
	if err != nil {
		return nil, introdb.MapGetMediaError(err)
	}

	return toIntroData(record), nil
}

// SkipIntroFrom returns the seek target to skip the intro from the current position.
func (m *Movie) SkipIntroFrom(position time.Duration) (time.Duration, error) {
	data, err := m.Intro()
	if err != nil {
		return 0, err
	}

	record := toMediaRecord(data)
	return introdb.IntroSkipTarget(record, position)
}

// CreditsStart estimates when credits begin.
func (m *Movie) CreditsStart(duration time.Duration) (time.Duration, bool) {
	data, err := m.Intro()
	if err != nil {
		return 0, false
	}

	record := toMediaRecord(data)
	return introdb.CreditsStart(record, duration.Milliseconds())
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

	root, err := m.client.febbox.ListFiles(shareKey, 0, "")
	if err != nil {
		return nil, err
	}

	direct := filesOnly(root)
	if len(direct) > 0 {
		m.file = &direct[0]
		return m.file, nil
	}

	seasons := seasonsOnly(root)
	if len(seasons) == 0 {
		return nil, nil
	}

	children, err := m.client.febbox.ListFiles(shareKey, seasons[0].FID, "")
	if err != nil {
		return nil, err
	}

	files := filesOnly(children)
	if len(files) == 0 {
		return nil, nil
	}

	m.file = &files[0]
	return m.file, nil
}