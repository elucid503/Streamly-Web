package vod

import (

	"mediakit/internal/febbox"
	"mediakit/internal/introdb"
	"mediakit/internal/meta"
	"mediakit/internal/quality"

)

// Deps is the interface that Client provides to Movie, Show, Season, and Episode handles.
type Deps interface {

	GetMovieDetails(id int) (meta.TitleDetails, error)
	GetShowDetails(id int) (meta.TitleDetails, error)
	GetEpisodeMeta(imdbID string, season, episode int) (EpisodeInfo, bool)
	GetSeasonEpisodes(imdbID string, season int) map[int]EpisodeInfo

	GetShowSeasonsByTMDB(tmdbID int) ([]ShowSeasonInfo, error)

	GetFebBoxID(id int, boxType int) (string, error)

	GetConsoleMovieFID(imdbID string) (int, error)
	GetConsoleEpisodeFID(imdbID string, season, episode int) (int, error)
	GetConsoleLinks(fid int) ([]febbox.Quality, error)

	ListFiles(shareKey string, parentID any, cookie string) ([]febbox.File, error)
	GetLinks(shareKey string, fid any, cookie string) ([]febbox.Quality, error)
	GetDownloadURL(shareKey string, fid any, cookie string) (string, error)

	GetIntro(query introdb.MediaQuery) (*introdb.MediaRecord, error)

	ResolveProviderStreams(tmdbID int, mediaType string, season, episode int) ([]quality.Quality, error)

}
