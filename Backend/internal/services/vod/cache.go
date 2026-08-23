package vod

import (
	"fmt"
	"sort"
	"sync"
	"time"

	mediakit "mediakit"

	"streamly/internal/services/upstream"

	"golang.org/x/sync/singleflight"
)

const (

	cacheTTL = 45 * time.Minute
	staleTTL = 24 * time.Hour

	maxEntries = 512

)

// SeasonDTO is a TV show season.
type SeasonDTO struct {

	Number int `json:"number"`
	Label string `json:"label"`

}

// EpisodeDTO is a single episode within a season.
type EpisodeDTO struct {

	Season int `json:"season"`
	Episode int `json:"episode"`

	Title string `json:"title"`
	Description string `json:"description,omitempty"`
	Poster string `json:"poster,omitempty"`

}

type cacheEntry[T any] struct {

	data T
	fetchedAt time.Time

}

// Cache caches show season and episode metadata.
type Cache struct {

	client *mediakit.Client

	mu sync.RWMutex
	group singleflight.Group

	seasons  map[int]cacheEntry[[]SeasonDTO]
	episodes map[string]cacheEntry[[]EpisodeDTO]

}

// New builds a Cache backed by client.
func New(client *mediakit.Client) *Cache {

	return &Cache{

		client: client,

		seasons: make(map[int]cacheEntry[[]SeasonDTO]),
		episodes: make(map[string]cacheEntry[[]EpisodeDTO]),

	}

}

// ShowSeasons returns cached or freshly fetched seasons for a show.
func (c *Cache) ShowSeasons(id int) ([]SeasonDTO, error) {

	c.mu.RLock()

	entry, ok := c.seasons[id]

	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < cacheTTL {

		return append([]SeasonDTO(nil), entry.data...), nil

	}

	result, err, _ := c.group.Do(fmt.Sprintf("seasons:%d", id), func() (any, error) {

		return upstream.Retry(3, func() ([]SeasonDTO, error) {

			return c.fetchShowSeasons(id)

		})

	})

	if err != nil {

		if ok && time.Since(entry.fetchedAt) < staleTTL {

			return append([]SeasonDTO(nil), entry.data...), nil

		}

		return nil, err

	}

	seasons := result.([]SeasonDTO)

	c.mu.Lock()

	c.pruneLocked()

	c.seasons[id] = cacheEntry[[]SeasonDTO]{data: seasons, fetchedAt: time.Now()}

	c.mu.Unlock()

	return append([]SeasonDTO(nil), seasons...), nil

}

// SeasonEpisodes returns cached or freshly fetched episodes for a season.
func (c *Cache) SeasonEpisodes(showID, season int) ([]EpisodeDTO, error) {

	key := fmt.Sprintf("%d:%d", showID, season)

	c.mu.RLock()

	entry, ok := c.episodes[key]

	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < cacheTTL {

		return append([]EpisodeDTO(nil), entry.data...), nil

	}

	result, err, _ := c.group.Do("episodes:"+key, func() (any, error) {

		return upstream.Retry(3, func() ([]EpisodeDTO, error) {

			return c.fetchSeasonEpisodes(showID, season)

		})

	})

	if err != nil {

		if ok && time.Since(entry.fetchedAt) < staleTTL {

			return append([]EpisodeDTO(nil), entry.data...), nil

		}

		return nil, err

	}

	episodes := result.([]EpisodeDTO)

	c.mu.Lock()

	c.pruneLocked()

	c.episodes[key] = cacheEntry[[]EpisodeDTO]{data: episodes, fetchedAt: time.Now()}

	c.mu.Unlock()

	return append([]EpisodeDTO(nil), episodes...), nil

}

func (c *Cache) pruneLocked() {

	now := time.Now()

	for id, entry := range c.seasons {

		if now.Sub(entry.fetchedAt) >= staleTTL || len(c.seasons) > maxEntries {

			delete(c.seasons, id)

		}

	}

	for key, entry := range c.episodes {

		if now.Sub(entry.fetchedAt) >= staleTTL || len(c.episodes) > maxEntries {

			delete(c.episodes, key)

		}

	}

}

func (c *Cache) fetchShowSeasons(id int) ([]SeasonDTO, error) {

	details, err := c.client.GetShowDetails(id)

	if err == nil && details.TMDBId > 0 {

		tmdbSeasons, tmdbErr := c.client.GetShowSeasonsByTMDB(details.TMDBId)

		if tmdbErr == nil && len(tmdbSeasons) > 0 {

			out := make([]SeasonDTO, 0, len(tmdbSeasons))

			for _, sn := range tmdbSeasons {

				label := sn.Name

				if label == "" {

					label = fmt.Sprintf("Season %d", sn.Number)

				}

				out = append(out, SeasonDTO{Number: sn.Number, Label: label})

			}

			return out, nil

		}

	}

	seasons, err := c.client.Show(id).Seasons()

	if err != nil {

		return nil, err

	}

	out := make([]SeasonDTO, len(seasons))

	for i, sn := range seasons {

		out[i] = SeasonDTO{Number: sn.Number(), Label: sn.Label()}

	}

	return out, nil

}

func (c *Cache) fetchSeasonEpisodes(showID, season int) ([]EpisodeDTO, error) {

	show := c.client.Show(showID)

	details, detailsErr := show.Details()

	if detailsErr == nil && details.IMDBId != "" {

		tmdbEps := c.client.GetSeasonEpisodes(details.IMDBId, season)

		if len(tmdbEps) > 0 {

			out := make([]EpisodeDTO, 0, len(tmdbEps))

			for num, info := range tmdbEps {

				out = append(out, EpisodeDTO{

					Season:  season,
					Episode: num,

					Title:       info.Title,
					Description: info.Description,

					Poster: info.Poster,
				})

			}

			sort.Slice(out, func(i, j int) bool {

				return out[i].Episode < out[j].Episode

			})

			return out, nil

		}

	}

	eps, err := show.Season(season).Episodes()

	if err != nil {

		return nil, err

	}

	numbers := make([]int, len(eps))

	for i, ep := range eps {

		numbers[i] = ep.Number()

	}

	var metaByEpisode map[int]mediakit.EpisodeInfo

	if detailsErr == nil && details.IMDBId != "" {

		metaByEpisode = show.EpisodeListInfo(season, numbers)

	}

	out := make([]EpisodeDTO, len(eps))

	for i, ep := range eps {

		info := metaByEpisode[ep.Number()]

		out[i] = EpisodeDTO{

			Season:  ep.SeasonNumber(),
			Episode: ep.Number(),

			Title:       info.Title,
			Description: info.Description,

			Poster: info.Poster,
		}

	}

	return out, nil

}
