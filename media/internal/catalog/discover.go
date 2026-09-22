package discover

import (
	"mediakit/internal/catalog/meta"
	"mediakit/internal/providers/showbox"
)

// Deps is the interface that Client provides to discovery functions and TopCategory.

type Deps interface {

	TopHot(mediaType showbox.MediaType, limit int) ([]string, error)

	TopLists(boxType showbox.BoxType) ([]showbox.TopList, error)

}

// TopCategory is a curated Showbox ranking list.

type TopCategory struct {

	id string

	name string

}

// ID returns the category identifier.

func (t *TopCategory) ID() string { return t.id }

// Name returns the display name.

func (t *TopCategory) Name() string { return t.name }

// Trending returns hot search keywords from Showbox.

func Trending(deps Deps, kind meta.MediaKind, limit int) ([]string, error) {

	mediaType := showbox.MediaMovie

	if kind == meta.MediaShow {

		mediaType = showbox.MediaTV

	}

	return deps.TopHot(mediaType, limit)

}

// TopCategories returns curated ranking categories for movies or TV.

func TopCategories(deps Deps, kind meta.MediaKind) ([]TopCategory, error) {

	boxType := showbox.BoxMovie

	if kind == meta.MediaShow {

		boxType = showbox.BoxSeries

	}

	lists, err := deps.TopLists(boxType)

	if err != nil {

		return nil, err

	}

	out := make([]TopCategory, len(lists))

	for i, list := range lists {

		out[i] = TopCategory{

			id: list.ID,

			name: list.DisplayName,

		}

	}

	return out, nil

}
