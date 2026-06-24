package vod

import (
	"testing"

	"mediakit/internal/febbox"
	"mediakit/internal/meta"
	"mediakit/internal/quality"
)

type providerMockDeps struct {

	mockDeps

	providerQualities []quality.Quality
	providerErr       error

}

func (m *providerMockDeps) ResolveProviderStreams(tmdbID int, mediaType string, season, episode int) ([]quality.Quality, error) {

	if m.providerErr != nil {

		return nil, m.providerErr

	}

	return m.providerQualities, nil

}

func (m *providerMockDeps) GetMovieDetails(id int) (meta.TitleDetails, error) {

	return meta.TitleDetails{

		TMDBId: 27205,
		IMDBId: "tt1375666",

	}, nil

}

func TestMovieQualitiesPrefersShareKey(t *testing.T) {

	deps := &providerMockDeps{

		providerQualities: []quality.Quality{

			{URL: "https://app.test/proxy/a", Label: "Vixsrc 1080p", Height: 1080, IsHLS: true},

		},

		mockDeps: mockDeps{

			shareKey: "share123",
			consoleMovieFID: 99,
			consoleLinks: map[int][]febbox.Quality{

				99: {{Quality: "1080p", URL: "https://febbox.test/console.mp4"}},

			},

			rootListing: map[any][]febbox.File{

				0: {{FID: 301, FileName: "Movie.1080p.mkv", IsDir: 0}},

			},

			links: map[int][]febbox.Quality{

				301: {{URL: "https://cdn.example/share.mp4", Quality: "1080p", Name: "HD"}},

			},

		},

	}

	movie := NewMovie(deps, 4059)

	qualities, err := movie.Qualities()

	if err != nil {

		t.Fatalf("Qualities: %v", err)

	}

	if len(qualities) != 1 || qualities[0].URL != "https://cdn.example/share.mp4" {

		t.Fatalf("expected share-key qualities, got %+v", qualities)

	}

}

func TestMovieQualitiesPrefersConsole(t *testing.T) {

	deps := &providerMockDeps{

		providerQualities: []quality.Quality{

			{URL: "https://app.test/proxy/a", Label: "Vixsrc 1080p", Height: 1080, IsHLS: true},
			{URL: "https://app.test/proxy/b", Label: "Vixsrc 720p", Height: 720, IsHLS: true},

		},

		mockDeps: mockDeps{

			consoleMovieFID: 99,
			consoleLinks: map[int][]febbox.Quality{

				99: {{Quality: "1080p", Name: "ORG", URL: "https://febbox.test/console.mp4"}},

			},

		},

	}

	movie := NewMovie(deps, 4059)

	qualities, err := movie.Qualities()

	if err != nil {

		t.Fatalf("Qualities: %v", err)

	}

	if len(qualities) != 1 || qualities[0].URL != "https://febbox.test/console.mp4" {

		t.Fatalf("expected console qualities, got %+v", qualities)

	}

}

func TestMovieQualitiesFallsBackToVixsrc(t *testing.T) {

	deps := &providerMockDeps{

		providerQualities: []quality.Quality{

			{URL: "https://app.test/proxy/a", Label: "Vixsrc 1080p", Height: 1080, IsHLS: true},
			{URL: "https://app.test/proxy/b", Label: "Vixsrc 720p", Height: 720, IsHLS: true},

		},

		mockDeps: mockDeps{

			consoleMovieFID: 0,

		},

	}

	movie := NewMovie(deps, 4059)

	qualities, err := movie.Qualities()

	if err != nil {

		t.Fatalf("Qualities: %v", err)

	}

	if len(qualities) != 2 || qualities[0].Height != 1080 {

		t.Fatalf("expected provider qualities, got %+v", qualities)

	}

}

func TestMovieQualitiesFallsBackToConsole(t *testing.T) {

	deps := &providerMockDeps{

		providerQualities: nil,
		mockDeps: mockDeps{

			consoleMovieFID: 99,
			consoleLinks: map[int][]febbox.Quality{

				99: {{Quality: "1080p", Name: "ORG", URL: "https://febbox.test/console.mp4"}},

			},

		},

	}

	movie := NewMovie(deps, 4059)

	qualities, err := movie.Qualities()

	if err != nil {

		t.Fatalf("Qualities: %v", err)

	}

	if len(qualities) == 0 || qualities[0].URL != "https://febbox.test/console.mp4" {

		t.Fatalf("expected console fallback, got %+v", qualities)

	}

}