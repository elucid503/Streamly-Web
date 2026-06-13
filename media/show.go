package mediakit

import (
	"fmt"
	"sync"

	"mediakit/internal/showbox"
)

// Show is a chainable handle for a TV series.
type Show struct {
	client *Client
	id     int

	mu       sync.Mutex
	details  *TitleDetails
	shareKey string
	shareErr error
	shareSet bool
}

// ID returns the Showbox catalogue id.
func (s *Show) ID() int {
	return s.id
}

// Details fetches and caches show metadata.
func (s *Show) Details() (TitleDetails, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.details != nil {
		return *s.details, nil
	}

	raw, err := s.client.showbox.GetShow(s.id)
	if err != nil {
		return TitleDetails{}, err
	}

	details := parseTitleDetails(raw)
	if details.IMDBId != "" {
		if meta, err := s.client.imdb.Series(details.IMDBId); err == nil {
			enrichTitleDetails(&details, meta)
		}
	}
	s.details = &details
	return details, nil
}

// EpisodeListInfo returns display metadata for a season's episodes in one pass.
func (s *Show) EpisodeListInfo(season int, episodeNumbers []int) map[int]EpisodeInfo {
	details, err := s.Details()
	if err != nil {
		return nil
	}

	out := make(map[int]EpisodeInfo, len(episodeNumbers))
	imdbEpisodes := map[int]EpisodeInfo{}
	if details.IMDBId != "" {
		for number, meta := range s.client.imdb.SeasonEpisodes(details.IMDBId, season) {
			imdbEpisodes[number] = EpisodeInfo{
				Title:       meta.Title,
				Description: meta.Description,
				Poster:      meta.Poster,
			}
		}
	}

	for _, number := range episodeNumbers {
		info := imdbEpisodes[number]
		if info.Title == "" && details.EpisodeTitles != nil {
			key := fmt.Sprintf("%d:%d", season, number)
			if title, ok := details.EpisodeTitles[key]; ok && title != "" {
				info.Title = title
			}
		}
		if info.Title == "" {
			info.Title = fmt.Sprintf("S%02dE%02d", season, number)
		}
		out[number] = info
	}

	return out
}

// ShareKey resolves the Febbox share key that hosts this show's files.
func (s *Show) ShareKey() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shareSet {
		return s.shareKey, s.shareErr
	}

	s.shareKey, s.shareErr = s.client.showbox.GetFebBoxID(s.id, showbox.BoxSeries)
	s.shareSet = true
	return s.shareKey, s.shareErr
}

// Seasons lists season folders inside the Febbox share.
func (s *Show) Seasons() ([]*Season, error) {
	shareKey, err := s.ShareKey()
	if err != nil {
		return nil, err
	}
	if shareKey == "" {
		return nil, fmt.Errorf("show %d: no febbox share key", s.id)
	}

	root, err := s.client.febbox.ListFiles(shareKey, 0, "")
	if err != nil {
		return nil, err
	}

	parsed := parseSeasons(root)
	seasons := make([]*Season, len(parsed))
	for i, item := range parsed {
		seasons[i] = &Season{
			show:   s,
			number: item.Number,
			label:  item.Label,
			folder: item.Folder,
		}
	}

	return seasons, nil
}

// Season returns a chainable handle for a specific season number.
func (s *Show) Season(number int) *Season {
	return &Season{show: s, number: number}
}

// Episode returns a chainable handle for a specific season and episode.
func (s *Show) Episode(season, episode int) *Episode {
	return &Episode{
		show:    s,
		season:  season,
		episode: episode,
	}
}

// NextEpisode returns the episode after the given season/episode, if one exists.
func (s *Show) NextEpisode(season, episode int) (*Episode, error) {
	seasons, err := s.Seasons()
	if err != nil {
		return nil, err
	}

	if len(seasons) == 0 {
		return s.nextFromFlatListing(episode)
	}

	for _, sn := range seasons {
		if sn.number != season {
			continue
		}

		eps, err := sn.Episodes()
		if err != nil {
			return nil, err
		}

		for _, ep := range eps {
			if ep.episode == episode+1 {
				return ep, nil
			}
		}
	}

	for _, sn := range seasons {
		if sn.number <= season {
			continue
		}

		eps, err := sn.Episodes()
		if err != nil {
			return nil, err
		}

		if len(eps) > 0 {
			return eps[0], nil
		}
	}

	return nil, nil
}

func (s *Show) nextFromFlatListing(currentEpisode int) (*Episode, error) {
	shareKey, err := s.ShareKey()
	if err != nil {
		return nil, err
	}

	root, err := s.client.febbox.ListFiles(shareKey, 0, "")
	if err != nil {
		return nil, err
	}

	eps := parseEpisodes(filesOnly(root), 1)
	for _, ep := range eps {
		if ep.Number == currentEpisode+1 {
			return &Episode{
				show:    s,
				season:  1,
				episode: ep.Number,
				file:    &ep.File,
			}, nil
		}
	}

	return nil, nil
}