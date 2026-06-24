package vod

import "mediakit/internal/quality"

func providerQualities(deps Deps, tmdbID int, mediaType string, season, episode int) ([]quality.Quality, bool) {

	if tmdbID <= 0 {

		return nil, false

	}

	qualities, err := deps.ResolveProviderStreams(tmdbID, mediaType, season, episode)

	if err != nil {

		streamDebugf("provider path failed tmdb:%d %s: %v", tmdbID, mediaType, err)

		return nil, false

	}

	if len(qualities) == 0 {

		streamDebugf("provider path empty tmdb:%d %s", tmdbID, mediaType)

		return nil, false

	}

	streamDebugf("provider path ok tmdb:%d %s count=%d", tmdbID, mediaType, len(qualities))

	return qualities, true

}