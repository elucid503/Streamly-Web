package providers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type vixsrcVariant struct {

	Height int
	URL    string

}

func parseVixsrcMasterPlaylist(body string) ([]vixsrcVariant, error) {

	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")

	lines := strings.Split(body, "\n")

	seen := make(map[int]vixsrcVariant)

	for i := 0; i < len(lines); i++ {

		line := strings.TrimSpace(lines[i])

		if !strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {

			continue

		}

		if strings.Contains(line, "TYPE=AUDIO") || strings.Contains(line, "TYPE=SUBTITLES") {

			continue

		}

		height := vixsrcVariantHeight(line)

		if height <= 0 {

			continue

		}

		urlLine := nextPlaylistURLLine(lines, i+1)

		if urlLine == "" {

			continue

		}

		if _, ok := seen[height]; ok {

			continue

		}

		seen[height] = vixsrcVariant{

			Height: height,
			URL:    urlLine,

		}

	}

	if len(seen) == 0 {

		return nil, fmt.Errorf("vixsrc: no video variants in playlist")

	}

	variants := make([]vixsrcVariant, 0, len(seen))

	for _, variant := range seen {

		variants = append(variants, variant)

	}

	sort.Slice(variants, func(i, j int) bool {

		return variants[i].Height > variants[j].Height

	})

	return variants, nil

}

func vixsrcVariantHeight(streamInfo string) int {

	match := vixResRE.FindStringSubmatch(streamInfo)

	if len(match) >= 2 {

		if height, err := strconv.Atoi(match[1]); err == nil && height > 0 {

			return height

		}

	}

	lower := strings.ToLower(streamInfo)

	for _, height := range []int{2160, 1080, 720, 480, 360} {

		if strings.Contains(lower, fmt.Sprintf("%dp", height)) {

			return height

		}

	}

	return 0

}

func nextPlaylistURLLine(lines []string, start int) string {

	for i := start; i < len(lines); i++ {

		line := strings.TrimSpace(lines[i])

		if line == "" || strings.HasPrefix(line, "#") {

			continue

		}

		return line

	}

	return ""

}