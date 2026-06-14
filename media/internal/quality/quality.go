package quality

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"mediakit/internal/febbox"
)

var (
	quality4KRE  = regexp.MustCompile(`(?i)2160|4k`)
	qualityPRe   = regexp.MustCompile(`(\d{3,4})\s*p`)
	qualityOrgRE = regexp.MustCompile(`(?i)org|origin`)
)

// Quality is one downloadable rendition of a video file.
type Quality struct {
	URL string

	Label string
	Name  string

	Speed string

	Size   string
	Height int

	IsHLS bool
}

// IsHLSURL reports whether a URL points at an HLS playlist.
func IsHLSURL(raw string) bool {

	lower := strings.ToLower(raw)
	path := strings.SplitN(lower, "?", 2)[0]

	if strings.HasSuffix(path, ".m3u8") || strings.HasSuffix(path, ".m3u") {

		return true

	}

	if strings.Contains(path, "/papi/tv/playlist/") || strings.Contains(path, "/api/proxy/playlist") {

		return true

	}

	return false

}

// IsWebPlayableURL reports whether a URL points at a browser-friendly container.
func IsWebPlayableURL(raw string) bool {

	path := strings.ToLower(strings.SplitN(strings.TrimSpace(raw), "?", 2)[0])

	switch {

	case strings.HasSuffix(path, ".mkv"), strings.HasSuffix(path, ".avi"), strings.HasSuffix(path, ".wmv"), strings.HasSuffix(path, ".flv"):

		return false

	default:

		return path != ""

	}

}

// ToQualities converts Febbox quality entries to the public Quality type.
func ToQualities(items []febbox.Quality) []Quality {

	out := make([]Quality, 0, len(items))

	for _, item := range items {

		label := item.Quality + " " + item.Name
		height := qualityHeight(label)

		out = append(out, Quality{

			URL: item.URL,

			Label: displayLabel(label, height),
			Name:  item.Name,

			Speed: item.Speed,

			Size:   item.Size,
			Height: height,

			IsHLS: IsHLSURL(item.URL),
		})

	}

	return out

}

// WithOriginalFallback appends the direct source file when Febbox only exposes
// a low-resolution transcode but the original file URL is still available.
func WithOriginalFallback(qualities []Quality, originalURL, fileName string) []Quality {

	originalURL = strings.TrimSpace(originalURL)

	if originalURL == "" || maxHeight(qualities) > 360 {

		return qualities

	}

	label := strings.TrimSpace("Original " + fileName)
	original := Quality{

		URL: originalURL,

		Label: "Highest Available",
		Name:  fileName,

		Height: qualityHeight(label),
		IsHLS:  IsHLSURL(originalURL),
	}

	if original.Height <= maxHeight(qualities) {

		return qualities

	}

	out := append([]Quality(nil), qualities...)
	out = append(out, original)

	return out

}

// NeedsOriginalFallback reports whether a direct source file lookup could add
// a useful higher-quality option to the current Febbox transcode list.
func NeedsOriginalFallback(qualities []Quality) bool {

	return maxHeight(qualities) <= 360

}

// PickQuality chooses the best source at or below targetHeight, preferring progressive files.
func PickQuality(qualities []Quality, targetHeight int) *Quality {

	qualities = preferProgressiveQualities(qualities)

	if len(qualities) == 0 {

		return nil

	}

	sorted := append([]Quality(nil), qualities...)

	sort.Slice(sorted, func(i, j int) bool {

		return sorted[i].Height < sorted[j].Height

	})

	if targetHeight <= 0 {

		return pickHighestKnown(sorted)

	}

	var best *Quality

	for i := range sorted {

		height := sorted[i].Height

		if height <= 0 {

			continue

		}

		if height <= targetHeight {

			best = &sorted[i]

		}

	}

	if best != nil {

		return best

	}

	return pickHighestKnown(sorted)

}

// PickNextLowerQuality returns the next lower rendition below belowHeight.
func PickNextLowerQuality(qualities []Quality, belowHeight int) *Quality {

	qualities = preferProgressiveQualities(qualities)

	if len(qualities) == 0 || belowHeight <= 0 {

		return nil

	}

	sorted := append([]Quality(nil), qualities...)

	sort.Slice(sorted, func(i, j int) bool {

		return sorted[i].Height > sorted[j].Height

	})

	for i := range sorted {

		height := sorted[i].Height

		if height > 0 && height < belowHeight {

			return &sorted[i]

		}

	}

	return nil

}

func maxHeight(qualities []Quality) int {

	max := 0

	for _, q := range qualities {

		if q.Height > max {

			max = q.Height

		}

	}

	return max

}

func displayLabel(raw string, height int) string {

	if height == 360 {

		return "Faster Streaming"

	}

	label := strings.TrimSpace(raw)

	if label == "" {

		return "Unknown Quality"

	}

	return label

}

func qualityHeight(label string) int {

	if quality4KRE.MatchString(label) {

		return 2160

	}

	if match := qualityPRe.FindStringSubmatch(label); len(match) > 1 {

		height, _ := strconv.Atoi(match[1])
		return height

	}

	if qualityOrgRE.MatchString(label) {

		if quality4KRE.MatchString(label) {

			return 2160

		}

		return 1080

	}

	return 0

}

func isMP4URL(raw string) bool {

	path := strings.ToLower(strings.SplitN(strings.TrimSpace(raw), "?", 2)[0])

	return strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, ".m4v")

}

func webPlayableQualities(qualities []Quality) []Quality {

	playable := make([]Quality, 0, len(qualities))

	for _, q := range qualities {

		if q.URL != "" && IsWebPlayableURL(q.URL) {

			playable = append(playable, q)

		}

	}

	if len(playable) > 0 {

		return playable

	}

	hls := make([]Quality, 0, len(qualities))

	for _, q := range qualities {

		if q.URL != "" && q.IsHLS {

			hls = append(hls, q)

		}

	}

	return hls

}

func preferProgressiveQualities(qualities []Quality) []Quality {

	qualities = webPlayableQualities(qualities)

	mp4 := make([]Quality, 0, len(qualities))
	otherProgressive := make([]Quality, 0, len(qualities))

	for _, q := range qualities {

		if q.URL == "" || q.IsHLS {

			continue

		}

		if isMP4URL(q.URL) {

			mp4 = append(mp4, q)

		} else {

			otherProgressive = append(otherProgressive, q)

		}

	}

	if len(mp4) > 0 {

		return mp4

	}

	if len(otherProgressive) > 0 {

		return otherProgressive

	}

	hls := make([]Quality, 0, len(qualities))

	for _, q := range qualities {

		if q.IsHLS && q.URL != "" {

			hls = append(hls, q)

		}

	}

	if len(hls) > 0 {

		return hls

	}

	return qualities

}

func pickHighestKnown(sorted []Quality) *Quality {

	for i := len(sorted) - 1; i >= 0; i-- {

		if sorted[i].Height > 0 {

			return &sorted[i]

		}

	}

	return &sorted[len(sorted)-1]

}
