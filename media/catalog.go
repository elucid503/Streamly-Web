package mediakit

import "mediakit/internal/tv"

// LiveCatalog wraps the live TV channel listing with search helpers.
type LiveCatalog struct {
	client *Client
	raw    *tv.ChannelCatalog
}

// Total returns the number of channels in the catalog.
func (c *LiveCatalog) Total() int {
	return c.raw.Total
}

// Channels returns all channels sorted by US popularity then name.
func (c *LiveCatalog) Channels() []*LiveChannel {
	sorted := c.raw.Sorted()
	out := make([]*LiveChannel, len(sorted))
	for i, channel := range sorted {
		out[i] = wrapChannel(c.client, channel)
	}
	return out
}

// FindByID returns a channel by daddyId.
func (c *LiveCatalog) FindByID(id string) (*LiveChannel, bool) {
	channel, ok := c.raw.FindByID(id)
	if !ok {
		return nil, false
	}
	return wrapChannel(c.client, channel), true
}

// FindBySlug returns a channel by slug.
func (c *LiveCatalog) FindBySlug(slug string) (*LiveChannel, bool) {
	channel, ok := c.raw.FindBySlug(slug)
	if !ok {
		return nil, false
	}
	return wrapChannel(c.client, channel), true
}

// FindByName returns a channel by exact name match.
func (c *LiveCatalog) FindByName(name string) (*LiveChannel, bool) {
	channel, ok := c.raw.FindByName(name)
	if !ok {
		return nil, false
	}
	return wrapChannel(c.client, channel), true
}

// Search finds channels whose name or slug contains the query.
func (c *LiveCatalog) Search(query string, limit int) []*LiveChannel {
	matches := c.raw.Search(query, limit)
	out := make([]*LiveChannel, len(matches))
	for i, channel := range matches {
		out[i] = wrapChannel(c.client, channel)
	}
	return out
}

// Filter returns channels matching optional country code and/or category.
func (c *LiveCatalog) Filter(countryCode, category string) []*LiveChannel {
	matches := c.raw.Filter(countryCode, category)
	out := make([]*LiveChannel, len(matches))
	for i, channel := range matches {
		out[i] = wrapChannel(c.client, channel)
	}
	return out
}

// PopularUS returns curated popular United States channels.
func (c *LiveCatalog) PopularUS(limit int) []*LiveChannel {
	matches := c.raw.PopularUS(limit)
	out := make([]*LiveChannel, len(matches))
	for i, channel := range matches {
		out[i] = wrapChannel(c.client, channel)
	}
	return out
}