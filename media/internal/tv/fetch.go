package tv

import (
	"log"
	"time"
)

// Warmup performs a synchronous first fetch, then starts periodic background refresh.
func (c *Client) Warmup() {

	if _, err := c.RefreshCatalog(); err != nil {

		log.Printf("[tv] initial catalog fetch failed: %v", err)

	}

	c.refreshOnce.Do(func() {

		go c.runCatalogRefreshLoop()

	})

	c.enrichmentOnce.Do(func() {

		go c.runCatalogEnrichmentLoop()

	})

}

func (c *Client) runCatalogRefreshLoop() {

	ticker := time.NewTicker(catalogTTL)
	defer ticker.Stop()

	for range ticker.C {

		if _, err := c.RefreshCatalog(); err != nil {

			log.Printf("[tv] catalog refresh failed: %v", err)

		}

	}

}
