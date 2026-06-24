package providers

import (
	"fmt"

	"mediakit/internal/quality"
)

// Provider resolves playable URLs for TMDB-identified content.
type Provider interface {

	Name() string
	Fetch(tmdbID int, mediaType string, season, episode int) ([]Stream, error)

}

// Resolver resolves streams through the configured primary provider (Vixsrc).
type Resolver struct {

	provider Provider

}

// New builds a Vixsrc resolver.
func New(_ string) *Resolver {

	return &Resolver{

		provider: newVixsrc(),

	}

}

// Resolve returns playable qualities from Vixsrc for the given TMDB title.
func (r *Resolver) Resolve(tmdbID int, mediaType string, season, episode int) ([]quality.Quality, error) {

	streams, err := r.provider.Fetch(tmdbID, mediaType, season, episode)

	if err != nil {

		return nil, err

	}

	qualities := make([]quality.Quality, 0, len(streams))

	for _, stream := range streams {

		if stream.URL == "" {

			continue

		}

		qualities = append(qualities, stream.ToQuality())

	}

	if len(qualities) == 0 {

		return nil, fmt.Errorf("providers: no streams found for tmdb:%d %s", tmdbID, mediaType)

	}

	return qualities, nil

}