package mediakit

import (
	"fmt"

	"mediakit/internal/febbox"
)

// Season is a chainable handle for one season of a TV show.
type Season struct {

	show *Show

	number int
	label string

	folder febbox.File

}

// Number returns the season number.
func (s *Season) Number() int {

	return s.number

}

// Label returns a human-readable season label.
func (s *Season) Label() string {

	if s.label != "" {

		return s.label

	}

	return fmt.Sprintf("Season %d", s.number)

}

// Episodes lists playable episodes in this season.
func (s *Season) Episodes() ([]*Episode, error) {

	shareKey, err := s.show.ShareKey()

	if err != nil {

		return nil, err

	}

	folder, err := s.resolveFolder(shareKey)

	if err != nil {

		return nil, err

	}

	children, err := s.show.client.febbox.ListFiles(shareKey, folder.FID, "")

	if err != nil {

		return nil, err

	}

	parsed := parseEpisodes(filesOnly(children), s.number)
	episodes := make([]*Episode, len(parsed))

	for i, item := range parsed {

		file := item.File

		episodes[i] = &Episode{

			show: s.show,

			season: s.number,
			episode: item.Number,

			file: &file,

		}

	}

	return episodes, nil

}

// Episode returns a chainable handle for a specific episode number.
func (s *Season) Episode(number int) *Episode {

	return &Episode{

		show: s.show,
		season: s.number,
		episode: number,

	}

}

func (s *Season) resolveFolder(shareKey string) (febbox.File, error) {

	if s.folder.FID != 0 {

		return s.folder, nil

	}

	root, err := s.show.client.febbox.ListFiles(shareKey, 0, "")

	if err != nil {

		return febbox.File{}, err

	}

	for _, item := range parseSeasons(root) {

		if item.Number == s.number {

			s.folder = item.Folder
			s.label = item.Label

			return s.folder, nil

		}

	}

	return febbox.File{}, fmt.Errorf("season %d not found", s.number)

}
