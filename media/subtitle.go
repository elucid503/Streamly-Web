package mediakit

import (
	"path/filepath"
	"regexp"
	"strings"

	"mediakit/internal/febbox"
)

var subtitleExt = map[string]string{
	".vtt": "vtt",
	".srt": "srt",
	".ass": "ass",
	".ssa": "ssa",
}

var langToken = regexp.MustCompile(`(?i)(english|en\.us|en\.gb|\ben\b|spanish|es\b|french|fr\b|german|de\b|italian|it\b|portuguese|pt\b)`)

// Subtitle describes an external subtitle file discovered alongside the video.
type Subtitle struct {

	ID string
	FID int

	Label string
	Language string

	Format string

	shareKey string

}

func (s Subtitle) ShareKey() string { return s.shareKey }

func isSubtitleName(name string) (string, bool) {

	ext := strings.ToLower(filepath.Ext(name))
	format, ok := subtitleExt[ext]

	return format, ok

}

func subtitleLabel(name string) string {

	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.ReplaceAll(base, ".", " ")
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.TrimSpace(base)

	if base == "" {

		return "Subtitles"

	}

	if match := langToken.FindString(base); match != "" {

		lower := strings.ToLower(match)
		return strings.ToUpper(lower[:1]) + lower[1:]

	}

	if len(base) > 42 {

		return base[:42] + "…"

	}

	return base

}

func subtitleLanguage(name string) string {

	match := langToken.FindString(name)

	if match == "" {

		return "und"

	}

	lower := strings.ToLower(match)

	switch {

		case strings.Contains(lower, "english"), lower == "en", strings.HasPrefix(lower, "en."):

			return "en"

		case strings.Contains(lower, "spanish"), lower == "es":

			return "es"

		case strings.Contains(lower, "french"), lower == "fr":

			return "fr"

		case strings.Contains(lower, "german"), lower == "de":

			return "de"

		case strings.Contains(lower, "italian"), lower == "it":

			return "it"

		case strings.Contains(lower, "portuguese"), lower == "pt":

			return "pt"

		default:

			return "und" // short for "undetermined" or "undefined"

	}

}

func subtitleFromFile(shareKey string, file febbox.File, format string) Subtitle {

	return Subtitle{

		ID: format + "-" + strings.ToLower(file.FileName),
		FID: file.FID,

		Label: subtitleLabel(file.FileName),
		Language: subtitleLanguage(file.FileName),

		Format: format,

		shareKey: shareKey,

	}

}

func collectSubtitles(shareKey string, siblings []febbox.File, video *febbox.File) []Subtitle {

	if video == nil {

		return nil

	}

	videoStem := strings.ToLower(strings.TrimSuffix(video.FileName, filepath.Ext(video.FileName)))
	out := make([]Subtitle, 0)

	for _, file := range siblings {

		if file.FID == video.FID {

			continue

		}

		format, ok := isSubtitleName(file.FileName)

		if !ok {

			continue

		}

		stem := strings.ToLower(strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName)))

		if videoStem != "" && !strings.Contains(stem, videoStem) && !strings.Contains(videoStem, stem) {

			// Still includes loose matches in the same folder, but we rank later.

		}

		out = append(out, subtitleFromFile(shareKey, file, format))

	}

	return out

}
