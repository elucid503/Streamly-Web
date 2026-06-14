package live

import "mediakit/internal/tv"

// Deps is the interface that Client provides to Channel and Catalog handles.
type Deps interface {

	ListChannels() (*tv.ChannelCatalog, error)
	ResolveStream(daddyID string) (tv.ResolvedStream, error)
	ResolveHLS(daddyID string) (string, error)

}

// ChannelInfo describes a live TV channel from the catalog.
type ChannelInfo struct {

	ID string
	DaddyID string

	Name string
	Slug string
	Logo string

	Country string

	Category string

	Status string

}

// Stream is a resolved live TV HLS playlist.
type Stream struct {

	URL string
	Referer string

	Channel ChannelInfo

}

// Catalog wraps the live TV channel listing with search helpers.
type Catalog struct {

	deps Deps
	raw *tv.ChannelCatalog

}

// NewCatalog wraps a ChannelCatalog with search helpers.
func NewCatalog(deps Deps, raw *tv.ChannelCatalog) *Catalog {

	return &Catalog{deps: deps, raw: raw}

}

// Total returns the number of channels in the catalog.
func (c *Catalog) Total() int {

	return c.raw.Total

}

// Channels returns all channels sorted by US popularity then name.
func (c *Catalog) Channels() []*Channel {

	sorted := c.raw.Sorted()
	out := make([]*Channel, len(sorted))

	for i, channel := range sorted {

		out[i] = wrapChannel(c.deps, channel)

	}

	return out

}

// FindByID returns a channel by daddyId.
func (c *Catalog) FindByID(id string) (*Channel, bool) {

	channel, ok := c.raw.FindByID(id)

	if !ok {

		return nil, false

	}

	return wrapChannel(c.deps, channel), true

}

// FindBySlug returns a channel by slug.
func (c *Catalog) FindBySlug(slug string) (*Channel, bool) {

	channel, ok := c.raw.FindBySlug(slug)

	if !ok {

		return nil, false

	}

	return wrapChannel(c.deps, channel), true

}

// FindByName returns a channel by exact name match.
func (c *Catalog) FindByName(name string) (*Channel, bool) {

	channel, ok := c.raw.FindByName(name)

	if !ok {

		return nil, false

	}

	return wrapChannel(c.deps, channel), true

}

// Search finds channels whose name or slug contains the query.
func (c *Catalog) Search(query string, limit int) []*Channel {

	matches := c.raw.Search(query, limit)
	out := make([]*Channel, len(matches))

	for i, channel := range matches {

		out[i] = wrapChannel(c.deps, channel)

	}

	return out

}

// Filter returns channels matching optional country code and/or category.
func (c *Catalog) Filter(countryCode, category string) []*Channel {

	matches := c.raw.Filter(countryCode, category)
	out := make([]*Channel, len(matches))

	for i, channel := range matches {

		out[i] = wrapChannel(c.deps, channel)

	}

	return out

}

// PopularUS returns curated popular United States channels.
func (c *Catalog) PopularUS(limit int) []*Channel {

	matches := c.raw.PopularUS(limit)
	out := make([]*Channel, len(matches))

	for i, channel := range matches {

		out[i] = wrapChannel(c.deps, channel)

	}

	return out

}

// Channel is a chainable handle for a live TV channel.
type Channel struct {

	deps Deps
	daddyID string
	info *tv.Channel

}

// NewChannel creates a Channel handle for the given daddyId.
func NewChannel(deps Deps, daddyID string) *Channel {

	return &Channel{deps: deps, daddyID: daddyID}

}

// DaddyID returns the channel's stream resolver id.
func (c *Channel) DaddyID() string {

	return c.daddyID

}

// Info returns catalog metadata for this channel.
func (c *Channel) Info() (ChannelInfo, error) {

	if c.info != nil {

		return channelInfo(*c.info), nil

	}

	catalog, err := c.deps.ListChannels()

	if err != nil {

		return ChannelInfo{}, err

	}

	channel, ok := catalog.FindByID(c.daddyID)

	if !ok {

		return ChannelInfo{

			DaddyID: c.daddyID,
			Name: c.daddyID,

		}, nil

	}

	c.info = &channel

	return channelInfo(channel), nil

}

// Resolve fetches the HLS playlist URL for this channel.
func (c *Channel) Resolve() (*Stream, error) {

	stream, err := c.deps.ResolveStream(c.daddyID)

	if err != nil {

		return nil, err

	}

	info, _ := c.Info()

	return &Stream{

		URL: stream.URL,

		Referer: stream.Referer,
		Channel: info,

	}, nil

}

// HLS returns just the resolved m3u8 URL.
func (c *Channel) HLS() (string, error) {

	return c.deps.ResolveHLS(c.daddyID)

}

func channelInfo(channel tv.Channel) ChannelInfo {

	return ChannelInfo{

		ID: channel.ID,
		DaddyID: channel.DaddyID,

		Name: channel.Name,
		Slug: channel.Slug,
		Logo: channel.Logo,

		Country: channel.Country.Code,
		Category: channel.Category,

		Status: channel.Status,

	}

}

func wrapChannel(deps Deps, channel tv.Channel) *Channel {

	ch := channel

	return &Channel{

		deps: deps,

		daddyID: channel.DaddyID,
		info: &ch,

	}

}
