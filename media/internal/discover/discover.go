package discover

import (
	"mediakit/internal/meta"
	"mediakit/internal/showbox"
)

// Deps is the interface that Client provides to discovery functions and TopCategory.
type Deps interface {

	TopHot(mediaType showbox.MediaType, limit int) ([]string, error)
	TopLists(boxType showbox.BoxType) ([]showbox.TopList, error)
	TopListMovies(listID string, page, limit int) ([]showbox.SearchResult, error)
	TopListTV(listID string, page, limit int) ([]showbox.SearchResult, error)

}

// TopCategory is a curated Showbox ranking list.
type TopCategory struct {

	deps Deps

	id string
	name string

	kind meta.MediaKind

}

// ID returns the category identifier.
func (t *TopCategory) ID() string { return t.id }

// Name returns the display name.
func (t *TopCategory) Name() string { return t.name }

// Titles returns titles in this ranking list.
func (t *TopCategory) Titles(page, limit int) ([]meta.SearchHit, error) {

	var results []showbox.SearchResult
	var err error

	if t.kind == meta.MediaMovie {

		results, err = t.deps.TopListMovies(t.id, page, limit)

	} else {

		results, err = t.deps.TopListTV(t.id, page, limit)

	}

	if err != nil {

		return nil, err

	}

	hits := make([]meta.SearchHit, len(results))

	for i, result := range results {

		hits[i] = meta.HitFromResult(result)

	}

	return hits, nil

}

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

			deps: deps,
			id: list.ID,
			name: list.DisplayName,
			kind: kind,

		}

	}

	return out, nil

}
