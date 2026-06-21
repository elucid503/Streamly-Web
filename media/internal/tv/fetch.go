package tv

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

//go:embed data/tv-channels.json
var embeddedCatalogJSON []byte // seed catalog loaded on startup; replaced by live scrape on first successful refresh

const catalogRefreshTimeout = 45 * time.Second

func seedEmbeddedCatalog(c *Client) {

	catalog, err := embeddedCatalog()

	if err != nil {

		log.Printf("[tv] embedded catalog unavailable: %v", err)
		return

	}

	c.storeCatalog(catalog)

}

// ListChannels returns the in-memory TV catalog without performing network I/O.
func (c *Client) ListChannels() (*ChannelCatalog, error) {

	if cached := c.anyCatalog(); cached != nil {

		return cached, nil

	}

	return nil, fmt.Errorf("tv: catalog unavailable")

}

// Warmup seeds the embedded catalog if needed and starts periodic background refresh.
func (c *Client) Warmup() {

	if c.anyCatalog() == nil {

		seedEmbeddedCatalog(c)

	}

	c.refreshOnce.Do(func() {

		go c.runCatalogRefreshLoop()

	})

}

func (c *Client) runCatalogRefreshLoop() {

	c.refreshCatalog()

	ticker := time.NewTicker(catalogTTL)
	defer ticker.Stop()

	for range ticker.C {

		c.refreshCatalog()

	}

}

func (c *Client) refreshCatalog() {

	if _, err := c.fetchCatalog(catalogRefreshTimeout); err != nil {

		log.Printf("[tv] catalog refresh failed: %v", err)

	}

}

func (c *Client) fetchCatalog(timeout time.Duration) (*ChannelCatalog, error) {

	client := &http.Client{Timeout: timeout}

	catalog, err := scrapeDLHDChannels(client)

	if err != nil {

		return nil, fmt.Errorf("dlhd.pk: %w", err)

	}

	c.storeCatalog(catalog)
	log.Printf("[tv] catalog refreshed from dlhd.pk (%d channels)", len(catalog.Channels))

	return catalog, nil

}

func embeddedCatalog() (*ChannelCatalog, error) {

	var catalog ChannelCatalog

	if err := json.Unmarshal(embeddedCatalogJSON, &catalog); err != nil {

		return nil, fmt.Errorf("decode embedded catalog: %w", err)

	}

	if len(catalog.Channels) == 0 {

		return nil, fmt.Errorf("embedded catalog is empty")

	}

	return &catalog, nil

}

func (c *Client) anyCatalog() *ChannelCatalog {

	c.catalogMu.RLock()
	defer c.catalogMu.RUnlock()

	if c.catalog == nil {

		return nil

	}

	copy := *c.catalog

	return &copy

}
