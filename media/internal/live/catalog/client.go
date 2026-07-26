package catalog

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	catalogTTL = 6 * time.Hour
)

// Client owns the metadata-only live channel catalog.
// It never resolves streams — that is live/source's job.
type Client struct {

	httpClient *http.Client

	mu sync.RWMutex
	catalog *Catalog

	refreshOnce sync.Once

}

// New builds a catalog Client.
func New() *Client {

	return &Client{

		httpClient: &http.Client{Timeout: fetchTimeout},

	}

}

// Warmup performs a synchronous first fetch, then starts periodic background refresh.
func (c *Client) Warmup() {

	if _, err := c.Refresh(); err != nil {

		log.Printf("[live/catalog] initial fetch failed: %v", err)

	}

	c.refreshOnce.Do(func() {

		go c.runRefreshLoop()

	})

}

// Refresh rebuilds the catalog from the metadata source.
func (c *Client) Refresh() (*Catalog, error) {

	channels, err := fetchIPTVCatalog(c.httpClient)

	if err != nil {

		return nil, err

	}

	if len(channels) == 0 {

		return nil, fmt.Errorf("live/catalog: empty catalog from metadata source")

	}

	cat := &Catalog{

		Channels: channels,
		FetchedAt: time.Now(),

	}

	c.store(cat)

	log.Printf("[live/catalog] refreshed %d channels", len(channels))

	return cat, nil

}

// List returns the in-memory catalog without network I/O.
func (c *Client) List() (*Catalog, error) {

	if cached := c.cached(); cached != nil {

		return cached, nil

	}

	return nil, fmt.Errorf("live/catalog: catalog unavailable")

}

func (c *Client) runRefreshLoop() {

	ticker := time.NewTicker(catalogTTL)
	defer ticker.Stop()

	for range ticker.C {

		if _, err := c.Refresh(); err != nil {

			log.Printf("[live/catalog] refresh failed: %v", err)

		}

	}

}

func (c *Client) store(catalog *Catalog) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.catalog = catalog

}

func (c *Client) cached() *Catalog {

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.catalog == nil {

		return nil

	}

	clone := *c.catalog
	clone.Channels = append([]Channel(nil), c.catalog.Channels...)

	return &clone

}
