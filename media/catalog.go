package mediakit

import (
	discover "mediakit/internal/catalog"
	"mediakit/internal/catalog/meta"
	"mediakit/internal/providers/tmdb"
)

// MediaKind distinguishes movies from TV series.
type MediaKind = meta.MediaKind

const (
	MediaMovie = meta.MediaMovie
	MediaShow = meta.MediaShow
)

// SearchHit is a single catalogue result from Showbox search.
type SearchHit = meta.SearchHit

// TitleDetails is user-facing metadata for a movie or show.
type TitleDetails = meta.TitleDetails

// TopCategory is a curated Showbox ranking list.
type TopCategory = discover.TopCategory

// TMDBClient fetches TMDB discover/trending lists.
type TMDBClient = tmdb.Client

// TMDBItem is a TMDB list result.
type TMDBItem = tmdb.Item

// TMDBKind is movie or tv for TMDB endpoints.
type TMDBKind = tmdb.MediaKind

const (
	TMDBMovie = tmdb.KindMovie
	TMDBTV = tmdb.KindTV
)

// TMDBGenreNames maps genre ids to display names.
func TMDBGenreNames(ids []int) []string {

	return tmdb.GenreNames(ids)

}
