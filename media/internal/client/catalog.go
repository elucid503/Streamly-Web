package client

import (
	discover "mediakit/internal/catalog"
	"mediakit/internal/catalog/meta"
	"mediakit/internal/providers/showbox"
	"mediakit/internal/providers/tmdb"
)

// TMDB returns the TMDB discover client.

func (c *Client) TMDB() *tmdb.Client {

	return c.tmdb

}

// SearchKind queries Showbox limited to movies or TV series.

func (c *Client) SearchKind(query string, kind meta.MediaKind) ([]meta.SearchHit, error) {

	mediaType := showbox.MediaMovie

	if kind == meta.MediaShow {

		mediaType = showbox.MediaTV

	}

	results, err := c.showbox.Search(query, mediaType, 1, 20)

	hits := make([]meta.SearchHit, 0, 20)

	appendHits := func(results []showbox.SearchResult, assumeKind meta.MediaKind) {

		for _, result := range results {

			hit := meta.HitFromResult(result)

			// Typed Search5 responses often omit box_type (0). Treat as the

			// requested kind rather than dropping every candidate.

			if hit.Kind == 0 {

				hit.Kind = assumeKind

			}

			if hit.Kind != kind {

				continue

			}

			hits = append(hits, hit)

		}

	}

	if err == nil {

		appendHits(results, kind)

	}

	// Typed TV search is unreliable on Showbox — always merge an untyped pass.

	if kind == meta.MediaShow || len(hits) == 0 {

		all, allErr := c.showbox.Search(query, showbox.MediaAll, 1, 20)

		if allErr == nil {

			appendHits(all, kind)

		}

	}

	seen := make(map[int]struct{}, len(hits))

	out := make([]meta.SearchHit, 0, len(hits))

	for _, hit := range hits {

		if _, ok := seen[hit.ID]; ok {

			continue

		}

		seen[hit.ID] = struct{}{}

		out = append(out, hit)

	}

	if len(out) == 0 && err != nil {

		return nil, err

	}

	return out, nil

}

// Warmup starts background live TV catalog refresh.

func (c *Client) Warmup() {

	c.liveCatalog.Warmup()

}

// Search queries the Showbox catalogue for movies and TV shows.

func (c *Client) Search(query string) ([]meta.SearchHit, error) {

	results, err := c.showbox.Search(query, showbox.MediaAll, 1, 25)

	if err != nil {

		return nil, err

	}

	hits := make([]meta.SearchHit, len(results))

	for i, result := range results {

		hits[i] = meta.HitFromResult(result)

	}

	return hits, nil

}

// Trending returns hot search keywords from Showbox.

func (c *Client) Trending(kind meta.MediaKind, limit int) ([]string, error) {

	return discover.Trending(c, kind, limit)

}

// TopCategories returns curated ranking categories for movies or TV.

func (c *Client) TopCategories(kind meta.MediaKind) ([]discover.TopCategory, error) {

	return discover.TopCategories(c, kind)

}

func (c *Client) TopHot(mediaType showbox.MediaType, limit int) ([]string, error) {

	return c.showbox.TopHot(mediaType, limit)

}

func (c *Client) TopLists(boxType showbox.BoxType) ([]showbox.TopList, error) {

	return c.showbox.TopLists(boxType)

}
