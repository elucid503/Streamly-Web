package introdb

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const minIntroDurationMs = 15_000

func viableIntros(segments []SegmentTimestamp) []SegmentTimestamp {

	var viable []SegmentTimestamp

	for _, segment := range segments {

		if segment.EndMs != nil {

			viable = append(viable, segment)

		}

	}

	if len(viable) == 0 {

		return nil

	}

	sort.Slice(viable, func(i, j int) bool {

		return viable[i].StartMs < viable[j].StartMs

	})

	for len(viable) > 1 && introDuration(viable[0]) < minIntroDurationMs {

		viable = viable[1:]

	}

	return viable

}

func introDuration(segment SegmentTimestamp) int64 {

	return *segment.EndMs - segment.StartMs

}

// QueryForTitle builds a lookup query from stored media metadata.
func QueryForTitle(tmdbID int, imdbID string, season, episode int, durationMs *int64) (MediaQuery, error) {

	if tmdbID <= 0 && strings.TrimSpace(imdbID) == "" {

		return MediaQuery{}, fmt.Errorf("introdb: no external id for intro lookup")

	}

	query := MediaQuery{

		TMDBId: tmdbID,
		IMDBId: strings.TrimSpace(imdbID),

		Season: season,
		Episode: episode,

	}

	if durationMs != nil {

		query.DurationMs = *durationMs

	}

	return query, nil

}

// MapGetMediaError normalizes transport and API failures for skip-intro handling.
func MapGetMediaError(err error) error {

	if err == nil {

		return nil

	}

	if errors.Is(err, ErrNotFound) {

		return ErrNoIntroData

	}

	return err

}
