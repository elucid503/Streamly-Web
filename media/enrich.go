package mediakit

import "mediakit/internal/imdb"

func enrichTitleDetails(details *TitleDetails, meta imdb.TitleMeta) {

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
