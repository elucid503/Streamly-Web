package captions

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reEpisodeSE    = regexp.MustCompile(`(?i)[._\s-]s(\d{1,2})[._\s-]?e(\d{1,3})[._\s-]`)
	reEpisodeXE    = regexp.MustCompile(`(?i)[._\s-](\d{1,2})x(\d{1,3})[._\s-]`)
	reEpisodeNumOnly = regexp.MustCompile(`(?i)[._\s-]e(\d{1,3})[._\s-]`)
)

// imdbQueryID strips the "tt" prefix so the bare integer can be passed to APIs
// that accept only numeric IMDB IDs (e.g. OpenSubtitles imdb_id parameter).
func imdbQueryID(s string) string {

	s = strings.TrimSpace(s)

	lower := strings.ToLower(s)

	if strings.HasPrefix(lower, "tt") {

		return s[2:]

	}

	return s

}

// normalizeFormat returns a canonical subtitle format identifier from an explicit
// format hint or from the filename extension. Returns "" when unrecognised.
func normalizeFormat(format, filename string) string {

	if format != "" {

		return canonicalFormat(format)

	}

	ext := strings.TrimPrefix(filepath.Ext(strings.ToLower(filename)), ".")

	return canonicalFormat(ext)

}

func canonicalFormat(s string) string {

	switch strings.ToLower(strings.TrimSpace(s)) {

	case "srt":

		return "srt"

	case "vtt", "webvtt":

		return "vtt"

	case "ass", "ssa":

		return "ass"

	case "sub":

		return "sub"

	case "zip":

		return "zip"

	default:

		return ""

	}

}

// nameMatchesEpisode reports whether filename appears to refer to the given
// season and episode numbers. If season == 0 only the episode number is matched.
func nameMatchesEpisode(name string, season, episode int) bool {

	padded := " " + name + " "

	if season > 0 {

		if m := reEpisodeSE.FindStringSubmatch(padded); m != nil {

			return parseInt(m[1]) == season && parseInt(m[2]) == episode

		}

		if m := reEpisodeXE.FindStringSubmatch(padded); m != nil {

			return parseInt(m[1]) == season && parseInt(m[2]) == episode

		}

		return false

	}

	if m := reEpisodeNumOnly.FindStringSubmatch(padded); m != nil {

		return parseInt(m[1]) == episode

	}

	return false

}

func parseInt(s string) int {

	var n int

	fmt.Sscanf(s, "%d", &n)

	return n

}
