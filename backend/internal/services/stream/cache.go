package stream

import (
	"fmt"
	"sync"
	"time"

	mediakit "mediakit"
)

const qualitiesTTL = 30 * time.Minute

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

	qualities, err := c.client.Movie(id).Qualities()

	if err != nil {

		if ok {

			return cloneQualities(entry.qualities), nil

		}

		return nil, err

	}

	c.mu.Lock()

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

	qualities, err := c.client.Show(showID).Episode(season, episode).Qualities()

	if err != nil {

		if ok {

			return cloneQualities(entry.qualities), nil

		}

		return nil, err

	}

	c.mu.Lock()

	c.episodes[key] = cacheEntry{

		qualities: cloneQualities(qualities),
		fetchedAt: time.Now(),

	}

	c.mu.Unlock()

	return qualities, nil

}

func cloneQualities(items []mediakit.Quality) []mediakit.Quality {

	if len(items) == 0 {

		return []mediakit.Quality{}

	}

	return append([]mediakit.Quality(nil), items...)

}
