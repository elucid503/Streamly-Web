package playback

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"streamly/internal/features/playback/captions"
)

func subtitleDataURI(content []byte, format string) string {

	var mimeType string

	switch strings.ToLower(strings.TrimSpace(format)) {

	case "vtt":

		mimeType = "text/vtt;charset=utf-8"

	default:

		mimeType = "text/plain;charset=utf-8"

	}

	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(content)

}

func friendlyLanguageName(code string) string {

	switch strings.ToLower(strings.TrimSpace(code)) {

	case "en", "eng", "english":

		return "English"

	case "es", "spa", "spanish":

		return "Spanish"

	case "fr", "fre", "french":

		return "French"

	case "de", "ger", "german":

		return "German"

	case "it", "ita", "italian":

		return "Italian"

	case "pt", "por", "portuguese":

		return "Portuguese"

	case "und", "":

		return "Unknown"

	default:

		if len(code) == 0 {

			return "Unknown"

		}

		if len(code) == 1 {

			return strings.ToUpper(code)

		}

		return strings.ToUpper(code[:1]) + strings.ToLower(code[1:])

	}

}

func friendlySubdlLabel(track captions.Track, langCount map[string]int) string {

	lang := friendlyLanguageName(track.Language)

	return disambiguateLabel(lang, langCount, track.Hi)

}

func friendlyOpenSubsLabel(track captions.Track, langCount map[string]int) string {

	lang := friendlyLanguageName(track.Language)

	return disambiguateLabel(lang, langCount, track.Hi)

}

func disambiguateLabel(lang string, langCount map[string]int, hearingImpaired bool) string {

	key := lang

	if hearingImpaired {

		key += "+sdh"

	}

	langCount[key]++

	count := langCount[key]

	label := lang

	if hearingImpaired {

		label += " (SDH)"

	}

	if count > 1 {

		label += fmt.Sprintf(" · Option %d", count)

	}

	return label

}

func subdlTrackID(track captions.Track) string {

	sum := sha256.Sum256([]byte(strings.TrimSpace(track.Path)))

	return "subdl-" + hex.EncodeToString(sum[:8])

}

func opensubsTrackID(track captions.Track) string {

	sum := sha256.Sum256([]byte(strings.TrimSpace(track.Path)))

	return "opensubs-" + hex.EncodeToString(sum[:8])

}
