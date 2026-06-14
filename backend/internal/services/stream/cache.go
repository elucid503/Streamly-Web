package stream

import (
	"fmt"
	"sync"
	"time"

	mediakit "mediakit"

	"golang.org/x/sync/singleflight"
)

const (

	qualitiesTTL = 30 * time.Minute
	maxEntries = 512

)

type cacheEntry struct {

	qualities []mediakit.Quality
	fetchedAt time.Time

}

// Cache caches resolved stream qualities for movies and episodes.
type Cache struct {

	client *mediakit.Client

	mu sync.RWMutex

	movies map[int]cacheEntry
	episodes map[string]cacheEntry

	group singleflight.Group

}

// New builds a Cache backed by client.
func New(client *mediakit.Client) *Cache {

	return &Cache{

		client: client,
		movies: make(map[int]cacheEntry),
		episodes: make(map[string]cacheEntry),

	}

}

// MovieQualities returns cached or freshly resolved qualities for a movie.
func (c *Cache) MovieQualities(id int) ([]mediakit.Quality, error) {

	c.mu.RLock()

	entry, ok := c.movies[id]

	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < qualitiesTTL {

		return cloneQualities(entry.qualities), nil

	}

	result, err, _ := c.group.Do(fmt.Sprintf("movie:%d", id), func() (any, error) {

		return c.client.Movie(id).Qualities()

	})

	if err != nil {

		if ok {

			return cloneQualities(entry.qualities), nil

		}

		return nil, err

	}

	qualities := result.([]mediakit.Quality)

	c.mu.Lock()

	c.pruneLocked()

	c.movies[id] = cacheEntry{

		qualities: cloneQualities(qualities),
		fetchedAt: time.Now(),

	}

	c.mu.Unlock()

	return qualities, nil

}

// EpisodeQualities returns cached or freshly resolved qualities for an episode.
func (c *Cache) EpisodeQualities(showID, season, episode int) ([]mediakit.Quality, error) {

	key := fmt.Sprintf("%d:%d:%d", showID, season, episode)

	c.mu.RLock()

	entry, ok := c.episodes[key]

	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < qualitiesTTL {

		return cloneQualities(entry.qualities), nil

	}

	result, err, _ := c.group.Do("episode:"+key, func() (any, error) {

		return c.client.Show(showID).Episode(season, episode).Qualities()

	})

	if err != nil {

		if ok {

			return cloneQualities(entry.qualities), nil

		}

		return nil, err

	}

	qualities := result.([]mediakit.Quality)

	c.mu.Lock()

	c.pruneLocked()

	c.episodes[key] = cacheEntry{

		qualities: cloneQualities(qualities),
		fetchedAt: time.Now(),

	}

	c.mu.Unlock()

	return qualities, nil

}

func (c *Cache) pruneLocked() {

	now := time.Now()

	for id, entry := range c.movies {

		if now.Sub(entry.fetchedAt) >= qualitiesTTL || len(c.movies) > maxEntries {

			delete(c.movies, id)

		}

	}

	for key, entry := range c.episodes {

		if now.Sub(entry.fetchedAt) >= qualitiesTTL || len(c.episodes) > maxEntries {

			delete(c.episodes, key)

		}

	}

}

func cloneQualities(items []mediakit.Quality) []mediakit.Quality {

	if len(items) == 0 {

		return []mediakit.Quality{}

	}

	return append([]mediakit.Quality(nil), items...)

}
