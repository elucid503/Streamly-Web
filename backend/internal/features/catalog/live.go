package catalog

import (
	"mediakit"
)

func (s *MediaService) LiveChannels() ([]LiveChannelDTO, error) {

	return s.catalog.LiveChannels(), nil

}

func (s *MediaService) LivePopular(limit int) ([]LiveChannelDTO, error) {

	return s.catalog.LivePopular(limit), nil

}

func (s *MediaService) LiveSearch(query string, limit int) ([]LiveChannelDTO, error) {

	return s.catalog.LiveSearch(query, limit), nil

}

func (s *MediaService) LiveChannel(id string) (LiveChannelDTO, bool) {

	if channel, ok := s.catalog.LiveChannel(id); ok {

		return channel, true

	}

	if name, ok := mediakit.DecodeSportsChannelID(id); ok {

		return LiveChannelDTO{

			ID: id,
			Name: name,
			Category: "Sports",
			Categories: []string{"Sports"},
			Enriched: true,

		}, true

	}

	return LiveChannelDTO{}, false

}

// LiveSports returns the cached sports matches — refreshed on its own
// background loop in catalogindex.Cache, same pattern as the live channel
// catalog, so this never blocks on a live upstream lookup.
func (s *MediaService) LiveSports() ([]SportsMatchDTO, error) {

	return s.catalog.SportsMatches(), nil

}

// RefreshLiveSports re-fetches the scoreboard so kickoff alerts use current startsAt and channel.
func (s *MediaService) RefreshLiveSports() []SportsMatchDTO {

	s.catalog.RefreshSportsNow()

	return s.catalog.SportsMatches()

}

// LiveSourceProviders returns anonymized live source options.
func (s *MediaService) LiveSourceProviders() []LiveSourceOption {

	raw := s.client.LiveSourceProviders()
	out := make([]LiveSourceOption, 0, len(raw))

	for _, p := range raw {

		out = append(out, LiveSourceOption{

			Key: p.Key,
			Label: p.Label,
			Description: p.Description,

		})

	}

	return out

}

// ResolveLiveStream returns a playable live stream for a catalog channel id.
// providerKey is an optional public anonymized key; empty means automatic.
func (s *MediaService) ResolveLiveStream(id string, providerKey string) (LiveStreamRef, error) {

	stream, err := s.client.ResolveChannelStream(id, providerKey)

	if err != nil {

		return LiveStreamRef{}, err

	}

	return LiveStreamRef{

		URL: stream.URL,
		IsHLS: stream.IsHLS || mediakit.IsHLSURL(stream.URL),
		Headers: stream.Headers,
		Provider: stream.Provider,

	}, nil

}
