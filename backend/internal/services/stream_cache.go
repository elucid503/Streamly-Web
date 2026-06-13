package services

import (
	"fmt"
	"time"

	mediakit "mediakit"
)

const streamQualitiesTTL = 30 * time.Minute

type qualitiesCacheEntry struct {

	qualities []mediakit.Quality
	fetchedAt time.Time

}

func (s *MediaService) cachedMovieQualities(id int) ([]mediakit.Quality, error) {

	s.qualitiesMu.RLock()

	entry, ok := s.movieQualitiesCache[id]

	s.qualitiesMu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < streamQualitiesTTL {

		return cloneQualities(entry.qualities), nil

	}

	qualities, err := s.client.Movie(id).Qualities()

	if err != nil {

		if ok {

			return cloneQualities(entry.qualities), nil

		}

		return nil, err

	}

	s.qualitiesMu.Lock()

	s.movieQualitiesCache[id] = qualitiesCacheEntry{

		qualities: cloneQualities(qualities),
		fetchedAt: time.Now(),

	}

	s.qualitiesMu.Unlock()

	return qualities, nil

}

func (s *MediaService) cachedEpisodeQualities(showID, season, episode int) ([]mediakit.Quality, error) {

	key := fmt.Sprintf("%d:%d:%d", showID, season, episode)

	s.qualitiesMu.RLock()

	entry, ok := s.episodeQualitiesCache[key]

	s.qualitiesMu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < streamQualitiesTTL {

		return cloneQualities(entry.qualities), nil

	}

	qualities, err := s.client.Show(showID).Episode(season, episode).Qualities()

	if err != nil {

		if ok {

			return cloneQualities(entry.qualities), nil

		}

		return nil, err

	}

	s.qualitiesMu.Lock()

	s.episodeQualitiesCache[key] = qualitiesCacheEntry{

		qualities: cloneQualities(qualities),
		fetchedAt: time.Now(),

	}

	s.qualitiesMu.Unlock()

	return qualities, nil

}

func cloneQualities(items []mediakit.Quality) []mediakit.Quality {

	if len(items) == 0 {

		return []mediakit.Quality{}

	}

	return append([]mediakit.Quality(nil), items...)

}
