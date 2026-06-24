package vod

import "mediakit/internal/quality"

func providerQualities(deps Deps, tmdbID int, mediaType string, season, episode int) ([]quality.Quality, bool) {

	if tmdbID <= 0 {

		return nil, false

	}

	qualities, err := deps.ResolveProviderStreams(tmdbID, mediaType, season, episode)

	if err != nil || len(qualities) == 0 {

		return nil, false

	}

	return qualities, true

}