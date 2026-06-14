package meta

import (
	"fmt"
	"strconv"
	"strings"

	"mediakit/internal/imdb"
	"mediakit/internal/showbox"
)

// MediaKind distinguishes movies from TV series.
type MediaKind int

const (

	MediaMovie MediaKind = 1
	MediaShow MediaKind = 2

)

// SearchHit is a single catalogue result from Showbox search.
type SearchHit struct {

	ID int
	Kind MediaKind

	Title string
	Year int
	Poster string
	Description string

	IMDBRating string

}

// TitleDetails is user-facing metadata for a movie or show.
type TitleDetails struct {

	Title string
	Year string

	Poster string
	Banner string
	Description string

	IMDBRating string

	TMDBId int
	IMDBId string

	EpisodeTitles map[string]string

}

// ParseTitleDetails converts a raw Showbox API payload into user-facing metadata.
func ParseTitleDetails(raw map[string]any) TitleDetails {

	text := func(key string) string {

		value, ok := raw[key]

		if !ok || value == nil || value == "" {

			return ""

		}

		return showbox.DecodeText(fmt.Sprint(value))

	}

	return TitleDetails{

		Title: fallback(text("title"), "Unknown title"),
		Year: text("year"),

		Poster: fallback(text("poster"), fallback(text("poster_org"), text("poster_min"))),
		Banner: fallback(text("banner"), fallback(text("backdrop"), fallback(text("cover"), text("still")))),

		Description: text("description"),

		IMDBRating: text("imdb_rating"),

		TMDBId: intFromAny(raw["tmdb_id"]),
		IMDBId: text("imdb_id"),

		EpisodeTitles: episodeTitleMap(raw),

	}

}

// EnrichTitleDetails overwrites fields with higher-quality data from the IMDb/Cinemeta catalog.
func EnrichTitleDetails(details *TitleDetails, meta imdb.TitleMeta) {

	if meta.Title != "" {

		details.Title = meta.Title

	}

	if meta.Year != "" {

		details.Year = meta.Year

	}

	if meta.Poster != "" {

		details.Poster = meta.Poster

	}

	if meta.Banner != "" {

		details.Banner = meta.Banner

	}

	if meta.Description != "" {

		details.Description = meta.Description

	}

	if meta.Rating != "" {

		details.IMDBRating = meta.Rating

	}

}

// HitFromResult converts a Showbox search result to a SearchHit.
func HitFromResult(result showbox.SearchResult) SearchHit {

	return SearchHit{

		ID: result.ID,
		Kind: MediaKind(result.BoxType),

		Title: result.Title,
		Year: result.Year,
		Poster: result.Poster,

		Description: result.Description,

		IMDBRating: result.IMDBRating,

	}

}

func episodeTitleMap(raw map[string]any) map[string]string {

	episodes, ok := raw["episode"].([]any)

	if !ok {

		return nil

	}

	titles := make(map[string]string)

	for _, item := range episodes {

		data, ok := item.(map[string]any)

		if !ok {

			continue

		}

		season, _ := data["season"].(float64)
		number, _ := data["episode"].(float64)

		title := showbox.DecodeText(fmt.Sprint(data["title"]))

		if season > 0 && number > 0 && title != "" {

			titles[fmt.Sprintf("%d:%d", int(season), int(number))] = title

		}

	}

	if len(titles) == 0 {

		return nil

	}

	return titles

}

func fallback(values ...string) string {

	for _, value := range values {

		if value != "" {

			return value

		}

	}

	return ""

}

func intFromAny(value any) int {

	switch typed := value.(type) {

		case int:

			return typed

		case int64:

			return int(typed)

		case float64:

			return int(typed)

		case string:

			parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
			return parsed

		default:

			return 0

	}

}
