package mediakit

import "mediakit/internal/showbox"

// Trending returns hot search keywords from Showbox.
func (c *Client) Trending(kind MediaKind, limit int) ([]string, error) {
	mediaType := showbox.MediaMovie
	if kind == MediaShow {
		mediaType = showbox.MediaTV
	}
	return c.showbox.TopHot(mediaType, limit)
}

// TopCategories returns curated ranking categories for movies or TV.
func (c *Client) TopCategories(kind MediaKind) ([]TopCategory, error) {
	boxType := showbox.BoxMovie
	if kind == MediaShow {
		boxType = showbox.BoxSeries
	}

	lists, err := c.showbox.TopLists(boxType)
	if err != nil {
		return nil, err
	}

	out := make([]TopCategory, len(lists))
	for i, list := range lists {
		out[i] = TopCategory{
			client: c,
			id:     list.ID,
			name:   list.DisplayName,
			kind:   kind,
		}
	}

	return out, nil
}

// TopCategory is a curated Showbox ranking list.
type TopCategory struct {
	client *Client
	id     string
	name   string
	kind   MediaKind
}

// ID returns the category identifier.
func (t *TopCategory) ID() string { return t.id }

// Name returns the display name.
func (t *TopCategory) Name() string { return t.name }

// Titles returns titles in this ranking list.
func (t *TopCategory) Titles(page, limit int) ([]SearchHit, error) {
	var results []showbox.SearchResult
	var err error

	if t.kind == MediaMovie {
		results, err = t.client.showbox.TopListMovies(t.id, page, limit)
	} else {
		results, err = t.client.showbox.TopListTV(t.id, page, limit)
	}

	if err != nil {
		return nil, err
	}

	hits := make([]SearchHit, len(results))
	for i, result := range results {
		hits[i] = hitFromResult(result)
	}

	return hits, nil
}