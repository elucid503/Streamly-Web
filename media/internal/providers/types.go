package providers

import (

	"fmt"
	"regexp"
	"strconv"
	"strings"

	"mediakit/internal/quality"

)

var streamHeightRE = regexp.MustCompile(`(\d{3,4})\s*p`)

// Stream is a playable video stream from a third-party embed provider.
type Stream struct {

	Name string
	URL string
	Quality string
	Provider string
	Headers map[string]string

	IsHLS bool

}

// ToQuality converts a Stream to a quality.Quality for use in the rest of the system.
func (s Stream) ToQuality() quality.Quality {

	height := parseStreamHeight(s.Quality)

	isHLS := s.IsHLS || quality.IsHLSURL(s.URL)

	// Adaptive HLS streams that don't declare a resolution are assumed 1080p
	// so the frontend quality picker can treat them as a valid selection.
	if height == 0 && isHLS {

		height = 1080

	}

	return quality.Quality{

		URL:   s.URL,
		Label: streamLabel(s.Provider, s.Quality, height),

		Height: height,
		IsHLS:  isHLS,

		Headers: s.Headers,

	}

}

func parseStreamHeight(q string) int {

	if match := streamHeightRE.FindStringSubmatch(strings.ToLower(q)); len(match) > 1 {

		h, _ := strconv.Atoi(match[1])
		return h

	}

	return 0

}

func streamLabel(provider, qual string, height int) string {

	if provider == "" {

		provider = "Provider"

	}

	q := strings.TrimSpace(qual)

	if q == "" || strings.EqualFold(q, "Auto") {

		return provider

	}

	var suffix string

	switch height {

	case 2160:
		suffix = " 4K"

	case 1080:
		suffix = " 1080p"

	case 720:
		suffix = " 720p"

	case 480:
		suffix = " 480p"

	case 360:
		suffix = " 360p"

	default:

		suffix = fmt.Sprintf(" %s", q)

	}

	return provider + suffix

}
