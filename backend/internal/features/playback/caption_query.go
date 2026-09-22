package playback

import (
	"fmt"

	"streamly/internal/features/playback/captions"
)

func (r *SubtitleResolver) movieCaptionQuery(id int) (captions.Query, error) {

	movie := r.media.Client().Movie(id)

	details, err := movie.Details()

	if err != nil {

		return captions.Query{}, err

	}

	query := captions.Query{

		IMDBId: details.IMDBId,
		TMDBId: details.TMDBId,

	}

	if file, err := movie.File(); err == nil && file != nil {

		query.VideoName = file.Name

	}

	if query.IMDBId == "" && query.TMDBId <= 0 {

		return captions.Query{}, fmt.Errorf("captions: no metadata ids for movie %d", id)

	}

	return query, nil

}

func (r *SubtitleResolver) episodeCaptionQuery(showID, season, episode int) (captions.Query, error) {

	show := r.media.Client().Show(showID)

	details, err := show.Details()

	if err != nil {

		return captions.Query{}, err

	}

	query := captions.Query{

		IMDBId: details.IMDBId,
		TMDBId: details.TMDBId,

		Season: season,
		Episode: episode,

	}

	if file, err := show.Episode(season, episode).File(); err == nil && file != nil {

		query.VideoName = file.Name

	}

	if query.IMDBId == "" && query.TMDBId <= 0 {

		return captions.Query{}, fmt.Errorf("captions: no metadata ids for show %d", showID)

	}

	return query, nil

}
