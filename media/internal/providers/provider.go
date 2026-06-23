package providers

import (

	"fmt"
	"sync"

	"mediakit/internal/quality"

)

// Provider is a streaming source that resolves playable URLs for TMDB-identified content.
type Provider interface {

	Name() string
	Fetch(tmdbID int, mediaType string, season, episode int) ([]Stream, error)

}

// Resolver runs all registered providers concurrently and aggregates their results.
type Resolver struct {

	providers []Provider

}

// New builds a Resolver with all available providers.
// Providers that need a TMDB API key are only registered when one is supplied.
func New(tmdbKey string) *Resolver {

	r := &Resolver{}

	r.providers = append(r.providers, newVixsrc())
	r.providers = append(r.providers, newVidlink())
	r.providers = append(r.providers, newVidsrc())

	if tmdbKey != "" {

		r.providers = append(r.providers, newVideasy(tmdbKey))
		r.providers = append(r.providers, newRive(tmdbKey))

	}

	return r

}

// Resolve queries all providers concurrently and returns aggregated qualities.
// Returns an error only when no provider returns any stream.
func (r *Resolver) Resolve(tmdbID int, mediaType string, season, episode int) ([]quality.Quality, error) {

	type result struct {

		streams []Stream

	}

	ch := make(chan result, len(r.providers))

	var wg sync.WaitGroup

	for _, p := range r.providers {

		wg.Add(1)

		go func(p Provider) {

			defer wg.Done()

			streams, err := p.Fetch(tmdbID, mediaType, season, episode)

			if err == nil {

				ch <- result{streams: streams}

			} else {

				ch <- result{}

			}

		}(p)

	}

	go func() {

		wg.Wait()
		close(ch)

	}()

	seen := make(map[string]struct{})
	var qualities []quality.Quality

	for res := range ch {

		for _, s := range res.streams {

			if s.URL == "" {

				continue

			}

			if _, dup := seen[s.URL]; dup {

				continue

			}

			seen[s.URL] = struct{}{}
			qualities = append(qualities, s.ToQuality())

		}

	}

	if len(qualities) == 0 {

		return nil, fmt.Errorf("providers: no streams found for tmdb:%d %s", tmdbID, mediaType)

	}

	return qualities, nil

}
