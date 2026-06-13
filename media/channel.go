package mediakit

import (
	"mediakit/internal/tv"
)

// LiveChannel is a chainable handle for a live TV channel.
type LiveChannel struct {
	client  *Client
	daddyID string
	info    *tv.Channel
}

// DaddyID returns the channel's stream resolver id.
func (c *LiveChannel) DaddyID() string {
	return c.daddyID
}

// Info returns catalog metadata for this channel.
func (c *LiveChannel) Info() (LiveChannelInfo, error) {
	if c.info != nil {
		return channelInfo(*c.info), nil
	}

	catalog, err := c.client.tv.ListChannels()
	if err != nil {
		return LiveChannelInfo{}, err
	}

	channel, ok := catalog.FindByID(c.daddyID)
	if !ok {
		return LiveChannelInfo{
			DaddyID: c.daddyID,
			Name:    c.daddyID,
		}, nil
	}

	c.info = &channel
	return channelInfo(channel), nil
}

// Resolve fetches the HLS playlist URL for this channel.
func (c *LiveChannel) Resolve() (*LiveStream, error) {
	stream, err := c.client.tv.ResolveStream(c.daddyID)
	if err != nil {
		return nil, err
	}

	info, _ := c.Info()

	return &LiveStream{
		URL:     stream.URL,
		Referer: stream.Referer,
		Channel: info,
	}, nil
}

// HLS returns just the resolved m3u8 URL.
func (c *LiveChannel) HLS() (string, error) {
	return c.client.tv.ResolveHLS(c.daddyID)
}

func channelInfo(channel tv.Channel) LiveChannelInfo {
	return LiveChannelInfo{
		ID:       channel.ID,
		DaddyID:  channel.DaddyID,
		Name:     channel.Name,
		Slug:     channel.Slug,
		Logo:     channel.Logo,
		Country:  channel.Country.Code,
		Category: channel.Category,
		Status:   channel.Status,
	}
}

func wrapChannel(client *Client, channel tv.Channel) *LiveChannel {
	ch := channel
	return &LiveChannel{
		client:  client,
		daddyID: channel.DaddyID,
		info:    &ch,
	}
}