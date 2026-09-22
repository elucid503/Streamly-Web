package client

import (
	"fmt"
	"time"

	"mediakit/internal/catalog/meta"
	"mediakit/internal/vod"
)

func (c *Client) cachedTitleDetails(items map[int]titleCacheEntry, id int) (meta.TitleDetails, bool) {

	c.titleMu.Lock()

	defer c.titleMu.Unlock()

	entry, ok := items[id]

	if !ok || time.Now().After(entry.expiry) {

		return meta.TitleDetails{}, false

	}

	return entry.details, true

}

func (c *Client) storeTitleDetails(items map[int]titleCacheEntry, id int, details meta.TitleDetails) {

	c.titleMu.Lock()

	defer c.titleMu.Unlock()

	items[id] = titleCacheEntry{details: details, expiry: time.Now().Add(titleDetailsTTL)}

}

func (c *Client) GetMovieDetails(id int) (meta.TitleDetails, error) {

	if details, ok := c.cachedTitleDetails(c.movieTitles, id); ok {

		return details, nil

	}

	result, err, _ := c.titleGroup.Do(fmt.Sprintf("movie:%d", id), func() (any, error) {

		if details, ok := c.cachedTitleDetails(c.movieTitles, id); ok {

			return details, nil

		}

		raw, err := c.showbox.GetMovie(id)

		if err != nil {

			return meta.TitleDetails{}, err

		}

		details := meta.ParseTitleDetails(raw)

		if details.IMDBId != "" {

			if m, err := c.imdb.Movie(details.IMDBId); err == nil {

				meta.EnrichTitleDetails(&details, m)

			}

		}

		c.storeTitleDetails(c.movieTitles, id, details)

		return details, nil

	})

	if err != nil {

		return meta.TitleDetails{}, err

	}

	return result.(meta.TitleDetails), nil

}

func (c *Client) GetShowDetails(id int) (meta.TitleDetails, error) {

	if details, ok := c.cachedTitleDetails(c.showTitles, id); ok {

		return details, nil

	}

	result, err, _ := c.titleGroup.Do(fmt.Sprintf("show:%d", id), func() (any, error) {

		if details, ok := c.cachedTitleDetails(c.showTitles, id); ok {

			return details, nil

		}

		raw, err := c.showbox.GetShow(id)

		if err != nil {

			return meta.TitleDetails{}, err

		}

		details := meta.ParseTitleDetails(raw)

		if details.IMDBId != "" {

			if m, err := c.imdb.Series(details.IMDBId); err == nil {

				meta.EnrichTitleDetails(&details, m)

			}

		}

		c.storeTitleDetails(c.showTitles, id, details)

		return details, nil

	})

	if err != nil {

		return meta.TitleDetails{}, err

	}

	return result.(meta.TitleDetails), nil

}

func (c *Client) GetEpisodeMeta(imdbID string, season, episode int) (vod.EpisodeInfo, bool) {

	m, ok := c.imdb.Episode(imdbID, season, episode)

	if !ok {

		return vod.EpisodeInfo{}, false

	}

	return vod.EpisodeInfo{

		Title: m.Title,

		Description: m.Description,

		Poster: m.Poster,

	}, true

}

func (c *Client) GetSeasonEpisodes(imdbID string, season int) map[int]vod.EpisodeInfo {

	episodes := c.imdb.SeasonEpisodes(imdbID, season)

	out := make(map[int]vod.EpisodeInfo, len(episodes))

	for number, m := range episodes {

		out[number] = vod.EpisodeInfo{

			Title: m.Title,

			Description: m.Description,

			Poster: m.Poster,

		}

	}

	return out

}

func (c *Client) GetShowSeasonsByTMDB(tmdbID int) ([]vod.ShowSeasonInfo, error) {

	summaries, err := c.imdb.ShowSeasonsByTMDBID(tmdbID)

	if err != nil {

		return nil, err

	}

	out := make([]vod.ShowSeasonInfo, len(summaries))

	for i, s := range summaries {

		out[i] = vod.ShowSeasonInfo{

			Number: s.Number,

			EpisodeCount: s.EpisodeCount,

			Name: s.Name,

		}

	}

	return out, nil

}
