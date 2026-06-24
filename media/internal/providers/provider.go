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

	if !vixsrcServerEnabled() {

		streamDebugf("vixsrc disabled for tmdb:%d %s", tmdbID, mediaType)

		return nil, fmt.Errorf("providers: vixsrc server resolution disabled")

	}

	streamDebugf("vixsrc resolve start tmdb:%d %s s%de%d", tmdbID, mediaType, season, episode)

	streams, err := r.provider.Fetch(tmdbID, mediaType, season, episode)

	if err != nil {

		streamDebugf("vixsrc resolve failed tmdb:%d %s: %v", tmdbID, mediaType, err)

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

		streamDebugf("vixsrc resolve empty streams tmdb:%d %s", tmdbID, mediaType)

		return nil, fmt.Errorf("providers: no streams found for tmdb:%d %s", tmdbID, mediaType)

	}

	streamDebugf("vixsrc resolve ok tmdb:%d %s qualities=%d", tmdbID, mediaType, len(qualities))

	return qualities, nil

}