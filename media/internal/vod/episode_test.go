package vod

import (
	"testing"

	"mediakit/internal/febbox"
	"mediakit/internal/introdb"
	"mediakit/internal/meta"
	"mediakit/internal/quality"
)

type mockDeps struct {

	shareKey string
	shareErr error

	consoleMovieFID int
	consoleEpisodeFID int
	consoleLinks map[int][]febbox.Quality

	rootListing map[any][]febbox.File
	links       map[int][]febbox.Quality
	linksErr    error

	downloadURLs map[int]string

	providerQualities []quality.Quality
	providerErr       error

}

func (m *mockDeps) GetMovieDetails(id int) (meta.TitleDetails, error) {

	return meta.TitleDetails{}, nil

}

func (m *mockDeps) GetShowDetails(id int) (meta.TitleDetails, error) {

	return meta.TitleDetails{IMDBId: "tt0000001", TMDBId: 1396}, nil

}

func (m *mockDeps) GetEpisodeMeta(imdbID string, season, episode int) (EpisodeInfo, bool) {

	return EpisodeInfo{}, false

}

func (m *mockDeps) GetSeasonEpisodes(imdbID string, season int) map[int]EpisodeInfo {

	return nil

}

func (m *mockDeps) GetShowSeasonsByTMDB(tmdbID int) ([]ShowSeasonInfo, error) {

	return nil, nil

}

func (m *mockDeps) GetFebBoxID(id int, boxType int) (string, error) {

	return m.shareKey, m.shareErr

}

func (m *mockDeps) GetConsoleMovieFID(imdbID string) (int, error) {

	return m.consoleMovieFID, nil

}

func (m *mockDeps) GetConsoleEpisodeFID(imdbID string, season, episode int) (int, error) {

	return m.consoleEpisodeFID, nil

}

func (m *mockDeps) GetConsoleLinks(fid int) ([]febbox.Quality, error) {

	if m.consoleLinks == nil {

		return nil, nil

	}

	return m.consoleLinks[fid], nil

}

func (m *mockDeps) ListFiles(shareKey string, parentID any, cookie string) ([]febbox.File, error) {

	if m.rootListing == nil {

		return nil, nil

	}

	return m.rootListing[parentID], nil

}

func (m *mockDeps) GetLinks(shareKey string, fid any, cookie string) ([]febbox.Quality, error) {

	if m.linksErr != nil {

		return nil, m.linksErr

	}

	id, _ := fid.(int)

	return m.links[id], nil

}

func (m *mockDeps) GetDownloadURL(shareKey string, fid any, cookie string) (string, error) {

	id, _ := fid.(int)

	return m.downloadURLs[id], nil

}

func (m *mockDeps) GetIntro(query introdb.MediaQuery) (*introdb.MediaRecord, error) {

	return nil, nil

}

func (m *mockDeps) ResolveProviderStreams(tmdbID int, mediaType string, season, episode int) ([]quality.Quality, error) {

	if m.providerErr != nil {

		return nil, m.providerErr

	}

	return m.providerQualities, nil

}

func TestEpisodeQualitiesReturnsLinksFromSeasonFolder(t *testing.T) {

	deps := &mockDeps{

		shareKey: "share123",

		rootListing: map[any][]febbox.File{

			0: {

				{FID: 100, FileName: "Season 1", IsDir: 1},
			},

			100: {

				{FID: 201, FileName: "Show.S01E01.1080p.mkv", IsDir: 0},
			},

		},

		links: map[int][]febbox.Quality{

			201: {

				{URL: "https://cdn.example/1080.mp4", Quality: "1080p", Name: "HD"},
			},

		},

	}

	show := NewShow(deps, 42)
	ep := show.Episode(1, 1)

	qualities, err := ep.Qualities()

	if err != nil {

		t.Fatalf("unexpected error: %v", err)

	}

	if len(qualities) == 0 {

		t.Fatal("expected qualities, got none")

	}

	if qualities[0].URL != "https://cdn.example/1080.mp4" {

		t.Fatalf("unexpected url: %s", qualities[0].URL)

	}

}

func TestEpisodeQualitiesEmptyWhenShareKeyMissing(t *testing.T) {

	deps := &mockDeps{shareKey: ""}

	show := NewShow(deps, 42)
	ep := show.Episode(1, 1)

	qualities, err := ep.Qualities()

	if err != nil {

		t.Fatalf("unexpected error: %v", err)

	}

	if len(qualities) != 0 {

		t.Fatalf("expected no qualities without share key, got %d", len(qualities))

	}

}

func TestEpisodeQualitiesSearchesAllSeasonFoldersOnMismatch(t *testing.T) {

	deps := &mockDeps{

		shareKey: "share123",

		rootListing: map[any][]febbox.File{

			0: {

				{FID: 100, FileName: "Season 1", IsDir: 1},
			},

			100: {

				{FID: 401, FileName: "Show.S02E01.720p.mkv", IsDir: 0},
			},

		},

		links: map[int][]febbox.Quality{

			401: {

				{URL: "https://cdn.example/s02e01.mp4", Quality: "720p", Name: "SD"},
			},

		},

	}

	show := NewShow(deps, 42)
	ep := show.Episode(2, 1)

	qualities, err := ep.Qualities()

	if err != nil {

		t.Fatalf("unexpected error: %v", err)

	}

	if len(qualities) == 0 {

		t.Fatal("expected qualities when episode file exists in a differently numbered season folder")

	}

}

func TestEpisodeQualitiesEmptyWhenSeasonFolderMissingAndRootHasOnlyDirs(t *testing.T) {

	deps := &mockDeps{

		shareKey: "share123",

		rootListing: map[any][]febbox.File{

			0: {

				{FID: 100, FileName: "Season 1", IsDir: 1},
				{FID: 200, FileName: "Season 2", IsDir: 1},
			},

		},

	}

	show := NewShow(deps, 42)
	ep := show.Episode(3, 1)

	qualities, err := ep.Qualities()

	if err != nil {

		t.Fatalf("unexpected error: %v", err)

	}

	if len(qualities) != 0 {

		t.Fatalf("expected no qualities when season folder is missing, got %d", len(qualities))

	}

}

func TestEpisodeQualitiesFlatRootListingFallback(t *testing.T) {

	deps := &mockDeps{

		shareKey: "share123",

		rootListing: map[any][]febbox.File{

			0: {

				{FID: 301, FileName: "Show.S01E01.720p.mkv", IsDir: 0},
				{FID: 302, FileName: "Show.S01E02.720p.mkv", IsDir: 0},
			},

		},

		links: map[int][]febbox.Quality{

			301: {

				{URL: "https://cdn.example/720.mp4", Quality: "720p", Name: "SD"},
			},

		},

	}

	show := NewShow(deps, 42)
	ep := show.Episode(1, 1)

	qualities, err := ep.Qualities()

	if err != nil {

		t.Fatalf("unexpected error: %v", err)

	}

	if len(qualities) == 0 {

		t.Fatal("expected qualities from flat root listing")

	}

}

func TestEpisodeQualitiesPrefersShareKeyOverVixsrc(t *testing.T) {

	deps := &mockDeps{

		providerQualities: []quality.Quality{

			{URL: "https://app.test/proxy/a", Label: "Vixsrc 1080p", Height: 1080, IsHLS: true},

		},

		shareKey: "share123",

		rootListing: map[any][]febbox.File{

			0: {{FID: 301, FileName: "Show.S01E01.720p.mkv", IsDir: 0}},

		},

		links: map[int][]febbox.Quality{

			301: {{URL: "https://cdn.example/720.mp4", Quality: "720p", Name: "SD"}},

		},

	}

	show := NewShow(deps, 42)
	ep := show.Episode(1, 1)

	qualities, err := ep.Qualities()

	if err != nil {

		t.Fatalf("unexpected error: %v", err)

	}

	if len(qualities) != 1 || qualities[0].URL != "https://cdn.example/720.mp4" {

		t.Fatalf("expected share-key qualities, got %+v", qualities)

	}

}

type febboxQualityErr string

func (e febboxQualityErr) Error() string {

	return string(e)

}

func TestEpisodeQualitiesFallsBackToVixsrc(t *testing.T) {

	deps := &mockDeps{

		providerQualities: []quality.Quality{

			{URL: "https://app.test/proxy/a", Label: "Vixsrc 1080p", Height: 1080, IsHLS: true},
			{URL: "https://app.test/proxy/b", Label: "Vixsrc 720p", Height: 720, IsHLS: true},

		},

	}

	show := NewShow(deps, 42)
	ep := show.Episode(1, 1)

	qualities, err := ep.Qualities()

	if err != nil {

		t.Fatalf("unexpected error: %v", err)

	}

	if len(qualities) != 2 || qualities[0].Height != 1080 {

		t.Fatalf("expected provider qualities, got %+v", qualities)

	}

}