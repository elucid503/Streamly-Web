package services

import (
	"fmt"
	"time"
)

const (
	vodCacheTTL  = 45 * time.Minute
	vodStaleTTL  = 24 * time.Hour
)

type vodCacheEntry[T any] struct {
	data      T
	fetchedAt time.Time
}

func (s *MediaService) cachedShowSeasons(id int) ([]SeasonDTO, error) {
	key := id

	s.vodMu.RLock()
	entry, ok := s.seasonsCache[key]
	s.vodMu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < vodCacheTTL {
		return append([]SeasonDTO(nil), entry.data...), nil
	}

	seasons, err := retryUpstream(3, func() ([]SeasonDTO, error) {
		s.throttleUpstream()
		return s.fetchShowSeasons(id)
	})

	if err != nil {
		if ok && time.Since(entry.fetchedAt) < vodStaleTTL {
			return append([]SeasonDTO(nil), entry.data...), nil
		}
		return nil, err
	}

	s.vodMu.Lock()
	s.seasonsCache[key] = vodCacheEntry[[]SeasonDTO]{data: seasons, fetchedAt: time.Now()}
	s.vodMu.Unlock()

	return append([]SeasonDTO(nil), seasons...), nil
}

func (s *MediaService) cachedSeasonEpisodes(showID, season int) ([]EpisodeDTO, error) {
	key := fmt.Sprintf("%d:%d", showID, season)

	s.vodMu.RLock()
	entry, ok := s.episodesCache[key]
	s.vodMu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < vodCacheTTL {
		return append([]EpisodeDTO(nil), entry.data...), nil
	}

	episodes, err := retryUpstream(3, func() ([]EpisodeDTO, error) {
		s.throttleUpstream()
		return s.fetchSeasonEpisodes(showID, season)
	})

	if err != nil {
		if ok && time.Since(entry.fetchedAt) < vodStaleTTL {
			return append([]EpisodeDTO(nil), entry.data...), nil
		}
		return nil, err
	}

	s.vodMu.Lock()
	s.episodesCache[key] = vodCacheEntry[[]EpisodeDTO]{data: episodes, fetchedAt: time.Now()}
	s.vodMu.Unlock()

	return append([]EpisodeDTO(nil), episodes...), nil
}

func (s *MediaService) fetchShowSeasons(id int) ([]SeasonDTO, error) {
	seasons, err := s.client.Show(id).Seasons()
	if err != nil {
		return nil, err
	}
	out := make([]SeasonDTO, len(seasons))
	for i, sn := range seasons {
		out[i] = SeasonDTO{Number: sn.Number(), Label: sn.Label()}
	}
	return out, nil
}

func (s *MediaService) fetchSeasonEpisodes(showID, season int) ([]EpisodeDTO, error) {
	show := s.client.Show(showID)
	eps, err := show.Season(season).Episodes()
	if err != nil {
		return nil, err
	}

	numbers := make([]int, len(eps))
	for i, ep := range eps {
		numbers[i] = ep.Number()
	}
	metaByEpisode := show.EpisodeListInfo(season, numbers)

	out := make([]EpisodeDTO, len(eps))
	for i, ep := range eps {
		info := metaByEpisode[ep.Number()]
		out[i] = EpisodeDTO{
			Season:      ep.SeasonNumber(),
			Episode:     ep.Number(),
			Title:       info.Title,
			Description: info.Description,
			Poster:      info.Poster,
		}
	}
	return out, nil
}