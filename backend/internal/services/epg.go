package services

import (
	mediakit "mediakit"
)

// ProgramDTO is a TV program airing slot.
type ProgramDTO struct {

	Title string `json:"title"`
	EpisodeTitle string `json:"episodeTitle,omitempty"`
	Summary string `json:"summary,omitempty"`

	StartsAt int64 `json:"startsAt"` // Unix seconds
	Runtime int `json:"runtime"`  // minutes

	Image string `json:"image,omitempty"`
	Season int `json:"season,omitempty"`
	Episode int `json:"episode,omitempty"`

	Genres []string `json:"genres,omitempty"`
	Rating string `json:"rating,omitempty"`
	Network string `json:"network,omitempty"`

}

// ChannelGuideEntry pairs a live channel with its current and next program.
type ChannelGuideEntry struct {

	Channel LiveChannelDTO `json:"channel"`
	Current *ProgramDTO `json:"current,omitempty"`
	Next *ProgramDTO `json:"next,omitempty"`
	Upcoming []ProgramDTO `json:"upcoming,omitempty"`

}

// LiveSchedule returns catalog channels with current and upcoming programs.
// Guide data is produced by mediakit's metadata-only guide layer (TVMaze).
func (s *MediaService) LiveSchedule() ([]ChannelGuideEntry, error) {

	entries, err := s.client.LiveSchedule()

	if err != nil {

		return nil, err

	}

	out := make([]ChannelGuideEntry, 0, len(entries))

	for _, e := range entries {

		out = append(out, ChannelGuideEntry{

			Channel: channelToDTO(e.Channel),
			Current: programToDTO(e.Current),
			Next: programToDTO(e.Next),
			Upcoming: programsToDTO(e.Upcoming),

		})

	}

	return out, nil

}

func channelToDTO(ch mediakit.Channel) LiveChannelDTO {

	return LiveChannelDTO{

		ID: ch.ID,
		Name: ch.Name,
		Slug: ch.Slug,
		Code: ch.Code,
		Logo: ch.Logo,

		Country: ch.Country.Code,
		CountryName: ch.Country.Name,
		Category: ch.Category,
		Categories: append([]string(nil), ch.Categories...),
		Network: ch.Network,
		Owners: append([]string(nil), ch.Owners...),
		Website: ch.Website,
		Enriched: ch.Enriched,

	}

}

func programToDTO(p *mediakit.Program) *ProgramDTO {

	if p == nil {

		return nil

	}

	return &ProgramDTO{

		Title: p.Title,
		EpisodeTitle: p.EpisodeTitle,
		Summary: p.Summary,

		StartsAt: p.StartsAt.Unix(),
		Runtime: p.Runtime,

		Image: p.Image,
		Season: p.Season,
		Episode: p.Episode,

		Genres: append([]string(nil), p.Genres...),
		Rating: p.Rating,
		Network: p.Network,

	}

}

func programsToDTO(items []mediakit.Program) []ProgramDTO {

	if len(items) == 0 {

		return nil

	}

	out := make([]ProgramDTO, 0, len(items))

	for i := range items {

		dto := programToDTO(&items[i])

		if dto != nil {

			out = append(out, *dto)

		}

	}

	return out

}
