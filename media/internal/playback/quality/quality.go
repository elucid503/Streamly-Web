package quality

import (
	"regexp"
	"strconv"
	"strings"

	"mediakit/internal/providers/febbox"
)

var (
	quality4KRE = regexp.MustCompile(`(?i)2160|4k`)
	qualityPRe = regexp.MustCompile(`(\d{3,4})\s*p`)
	qualityOrgRE = regexp.MustCompile(`(?i)org|origin`)
)

// Quality is one downloadable rendition of a video file.
type Quality struct {

	URL string

	Label string
	Name string

	Speed string

	Size string
	Height int

	IsHLS bool

	// Headers are optional playback headers required by some stream providers (e.g. Referer).
	Headers map[string]string

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
			Name: item.Name,

			Speed: item.Speed,

			Size: item.Size,
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

	if originalURL == "" || maxHeight(qualities) > 360 || !IsWebPlayableURL(originalURL) {

		return qualities

	}

	label := strings.TrimSpace("Original " + fileName)
	original := Quality{

		URL: originalURL,

		Label: "Highest Available",
		Name: fileName,

		Height: qualityHeight(label),
		IsHLS: IsHLSURL(originalURL),

	}

	if original.Height <= maxHeight(qualities) {

		return qualities

	}

	out := append([]Quality(nil), qualities...)
	out = append(out, original)

	return out

}

// WithOriginalAtHeight replaces the transcode at height with the original source file URL when that source can be played directly in the browser.
func WithOriginalAtHeight(qualities []Quality, originalURL, fileName string, height int) []Quality {

	originalURL = strings.TrimSpace(originalURL)

	if originalURL == "" || height <= 0 || !IsWebPlayableURL(originalURL) || !IsWebPlayableURL(fileName) {

		return qualities

	}

	original := Quality{

		URL: originalURL,

		Label: "Original Format",
		Name: fileName,

		Height: height,
		IsHLS: IsHLSURL(originalURL),

	}

	out := make([]Quality, 0, len(qualities)+1)
	replaced := false

	for _, item := range qualities {

		if item.Height == height {

			if !replaced {

				out = append(out, original)
				replaced = true

			}

			continue

		}

		out = append(out, item)

	}

	if !replaced {

		out = append(out, original)

	}

	return out

}

// NeedsOriginalFallback reports whether a direct source file lookup could add a useful higher-quality option to the current Febbox transcode list.
func NeedsOriginalFallback(qualities []Quality) bool {

	return maxHeight(qualities) <= 360

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

	switch height {

	case 360:

		return "Faster Streaming"

	case 720:

		return "Balanced Quality"

	case 1080:

		return "Uses More Data"

	case 2160:

		return "Best Available"

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
