package catalog

import (
	"fmt"
	"time"

	"mediakit"
)

func (s *MediaService) MovieDetails(id int) (*TitleDetailsDTO, error) {

	if details, ok := s.cachedTitleDetails(s.movieDetails, id); ok {

		return details, nil

	}

	result, err, _ := s.detailsGroup.Do(fmt.Sprintf("movie:%d", id), func() (any, error) {

		details, err := s.client.Movie(id).Details()

		if err != nil {

			return nil, err

		}

		dto := titleDetailsToDTO(id, "movie", details)

		s.setTitleDetails(true, id, dto)

		return dto, nil

	})

	if err != nil {

		return nil, err

	}

	return cloneTitleDetails(result.(*TitleDetailsDTO)), nil

}

func (s *MediaService) ShowDetails(id int) (*TitleDetailsDTO, error) {

	if details, ok := s.cachedTitleDetails(s.showDetails, id); ok {

		return details, nil

	}

	result, err, _ := s.detailsGroup.Do(fmt.Sprintf("show:%d", id), func() (any, error) {

		details, err := s.client.Show(id).Details()

		if err != nil {

			return nil, err

		}

		dto := titleDetailsToDTO(id, "show", details)

		s.setTitleDetails(false, id, dto)

		return dto, nil

	})

	if err != nil {

		return nil, err

	}

	return cloneTitleDetails(result.(*TitleDetailsDTO)), nil

}

func (s *MediaService) ShowSeasons(id int) ([]SeasonDTO, error) {

	return s.vod.ShowSeasons(id)

}

func (s *MediaService) SeasonEpisodes(showID, season int) ([]EpisodeDTO, error) {

	return s.vod.SeasonEpisodes(showID, season)

}

func (s *MediaService) EpisodeDetails(showID, season, episode int) (*EpisodeDTO, error) {

	ep := s.client.Show(showID).Episode(season, episode)

	info, err := ep.Info()

	if err != nil {

		return nil, err

	}

	return &EpisodeDTO{

		Season: season,
		Episode: episode,

		Title: info.Title,
		Description: info.Description,
		Poster: info.Poster,

	}, nil

}

func (s *MediaService) MovieQualities(id int) ([]mediakit.Quality, error) {

	return s.stream.MovieQualities(id)

}

func (s *MediaService) EpisodeQualities(showID, season, episode int) ([]mediakit.Quality, error) {

	return s.stream.EpisodeQualities(showID, season, episode)

}

func (s *MediaService) MovieIntro(id int, durationMs int64) (*IntroDTO, error) {

	movie := s.client.Movie(id)

	var opts []mediakit.IntroOption

	if durationMs > 0 {

		opts = append(opts, mediakit.WithDuration(time.Duration(durationMs)*time.Millisecond))

	}

	data, err := movie.Intro(opts...)

	if err != nil {

		return nil, err

	}

	return introToDTO(data, func(d time.Duration) (time.Duration, bool) {

		return movie.CreditsStart(d)

	}), nil

}

func (s *MediaService) EpisodeIntro(showID, season, episode int, durationMs int64) (*IntroDTO, error) {

	ep := s.client.Show(showID).Episode(season, episode)

	var opts []mediakit.IntroOption

	if durationMs > 0 {

		opts = append(opts, mediakit.WithDuration(time.Duration(durationMs)*time.Millisecond))

	}

	data, err := ep.Intro(opts...)

	if err != nil {

		return nil, err

	}

	return introToDTO(data, func(d time.Duration) (time.Duration, bool) {

		return ep.CreditsStart(d)

	}), nil

}

func (s *MediaService) NextEpisode(showID, season, episode int) (*NextEpisodeDTO, error) {

	details, err := s.client.GetShowDetails(showID)

	if err != nil {

		return nil, err

	}

	if details.TMDBId <= 0 {

		return nil, nil

	}

	tmdbSeasons, err := s.client.GetShowSeasonsByTMDB(details.TMDBId)

	if err != nil {

		return nil, err

	}

	var currentEpisodeCount, nextSeasonNum int

	for _, sn := range tmdbSeasons {

		if sn.Number == season {

			currentEpisodeCount = sn.EpisodeCount

		}

		if sn.Number > season && (nextSeasonNum == 0 || sn.Number < nextSeasonNum) {

			nextSeasonNum = sn.Number

		}

	}

	var nextSzn, nextEp int

	if episode < currentEpisodeCount {

		nextSzn, nextEp = season, episode+1

	} else if nextSeasonNum > 0 {

		nextSzn, nextEp = nextSeasonNum, 1

	} else {

		return nil, nil

	}

	var title string

	if details.IMDBId != "" {

		if info, ok := s.client.GetEpisodeMeta(details.IMDBId, nextSzn, nextEp); ok {

			title = info.Title

		}

	}

	return &NextEpisodeDTO{

		Season: nextSzn,
		Episode: nextEp,
		Title: title,

	}, nil

}

func (s *MediaService) cachedTitleDetails(items map[int]titleDetailsCacheEntry, id int) (*TitleDetailsDTO, bool) {

	s.detailsMu.RLock()

	entry, ok := items[id]

	s.detailsMu.RUnlock()

	if !ok || time.Since(entry.fetchedAt) >= titleDetailsTTL {

		return nil, false

	}

	return cloneTitleDetails(entry.details), true

}

func (s *MediaService) setTitleDetails(movie bool, id int, details *TitleDetailsDTO) {

	s.detailsMu.Lock()

	defer s.detailsMu.Unlock()

	s.pruneTitleDetailsLocked()

	entry := titleDetailsCacheEntry{

		details: cloneTitleDetails(details),
		fetchedAt: time.Now(),

	}

	if movie {

		s.movieDetails[id] = entry

	} else {

		s.showDetails[id] = entry

	}

}

func (s *MediaService) pruneTitleDetailsLocked() {

	now := time.Now()

	for id, entry := range s.movieDetails {

		if now.Sub(entry.fetchedAt) >= titleDetailsTTL || len(s.movieDetails) > titleDetailsMaxEntries {

			delete(s.movieDetails, id)

		}

	}

	for id, entry := range s.showDetails {

		if now.Sub(entry.fetchedAt) >= titleDetailsTTL || len(s.showDetails) > titleDetailsMaxEntries {

			delete(s.showDetails, id)

		}

	}

}
