package catalog

import (
	"sort"
	"time"

	"mediakit"
)

// QualitiesToDTO converts a slice of qualities to DTOs, filtering to web-playable URLs and sorting by height descending.
func QualitiesToDTO(items []mediakit.Quality) []QualityDTO {

	out := make([]QualityDTO, 0, len(items))

	for _, q := range items {

		if q.URL == "" || !mediakit.IsWebPlayableURL(q.URL) {

			continue

		}

		dto := QualityDTO{

			Label: q.Label,
			Height: q.Height,
			IsHLS: q.IsHLS,
			URL: q.URL,
			Headers: cloneQualityHeaders(q.Headers),

		}

		out = append(out, dto)

	}

	sort.Slice(out, func(i, j int) bool {

		return out[i].Height > out[j].Height

	})

	return out

}

// BuildStreamDTO assembles a StreamDTO from all available qualities.
func BuildStreamDTO(qualities []mediakit.Quality) *StreamDTO {

	dtos := QualitiesToDTO(qualities)

	if len(dtos) == 0 {

		return nil

	}

	return &StreamDTO{

		Qualities: dtos,

	}

}

func titleDetailsToDTO(id int, kind string, details mediakit.TitleDetails) *TitleDetailsDTO {

	return &TitleDetailsDTO{

		ID: id,
		Kind: kind,

		Title: details.Title,
		Year: details.Year,

		Poster: details.Poster,
		Banner: details.Banner,

		Description: details.Description,
		Rating: details.IMDBRating,

	}

}

func cloneQualityHeaders(headers map[string]string) map[string]string {

	if len(headers) == 0 {

		return nil

	}

	out := make(map[string]string, len(headers))

	for key, value := range headers {

		if value == "" {

			continue

		}

		out[key] = value

	}

	return out

}

func cloneTitleDetails(details *TitleDetailsDTO) *TitleDetailsDTO {

	if details == nil {

		return nil

	}

	cp := *details

	return &cp

}

func introToDTO(data *mediakit.IntroData, creditsFn func(time.Duration) (time.Duration, bool)) *IntroDTO {

	if data == nil {

		return &IntroDTO{}

	}

	dto := &IntroDTO{}

	if start, end, ok := data.IntroWindow(); ok {

		startMs := start.Milliseconds()
		endMs := end.Milliseconds()

		dto.IntroStartMs = &startMs
		dto.IntroEndMs = &endMs

	}

	if creditsFn != nil {

		if credits, ok := creditsFn(0); ok {

			ms := credits.Milliseconds()

			dto.CreditsStartMs = &ms

		}

	}

	return dto

}
