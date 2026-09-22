package client

import (
	"context"
	"fmt"

	"mediakit/internal/live/catalog"
	"mediakit/internal/live/guide"
	"mediakit/internal/live/source"
	livesports "mediakit/internal/live/sports"
)

// Channels returns the current live TV channel catalog (metadata only).
func (c *Client) Channels() (*catalog.Catalog, error) {

	return c.liveCatalog.List()

}

// ResolveChannelStream resolves a full stream reference (URL + headers) for a channel.
// providerKey is an optional public anonymized key ("auto", "s1", …); empty means auto.
func (c *Client) ResolveChannelStream(channelID string, providerKey ...string) (source.Stream, error) {

	if name, ok := livesports.DecodeChannelID(channelID); ok {

		// Synthetic sports identities are created only after confirming an NTV
		// inventory match. Keep them pinned to that provider so another fuzzy
		// provider cannot resolve the team name to an unrelated channel.
		return c.liveSource.ResolveWith(context.Background(), source.Request{

			ChannelID: channelID,
			Name: name,

		}, "s2")

	}

	cat, err := c.liveCatalog.List()

	if err != nil {

		return source.Stream{}, err

	}

	ch, ok := cat.FindByID(channelID)

	if !ok {

		return source.Stream{}, fmt.Errorf("live: channel %q not found", channelID)

	}

	key := ""

	if len(providerKey) > 0 {

		key = providerKey[0]

	}

	return c.liveSource.ResolveWith(context.Background(), source.Request{

		ChannelID: ch.ID,
		Name: ch.Name,
		AltNames: append([]string(nil), ch.AltNames...),
		Network: ch.Network,
		Country: ch.Country.Code,

	}, key)

}

// LiveSourceProviders returns anonymized live source options for the client UI.
func (c *Client) LiveSourceProviders() []source.PublicProvider {

	return c.liveSource.PublicProviders()

}

// LiveSchedule returns guide entries for popular channels.
func (c *Client) LiveSchedule() ([]guide.Entry, error) {

	return c.liveGuide.Schedule()

}

// Matches returns current and upcoming sports fixtures from scoreboard sources.
func (c *Client) Matches() ([]livesports.Match, error) {

	return c.liveSports.Matches()

}
