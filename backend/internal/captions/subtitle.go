package captions

import (
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Track describes a remote subtitle candidate.
type Track struct {

	Path string
	Name string

	Language string
	Format string

	Hi bool

}

func extractSubtitle(data []byte, season, episode int) ([]byte, string, error) {

	if len(data) >= 4 && data[0] == 'P' && data[1] == 'K' {

		payload, format, err := extractFromZip(data, season, episode)

		if err != nil {

			return nil, "", err

		}

		return normalizeSubtitleUTF8(payload), format, nil

	}

	if looksLikeSubtitle(data) && looksEnglishSubtitle(data) {

		return normalizeSubtitleUTF8(data), detectFormat(data), nil

	}

	return nil, "", ErrNoSubtitle

}

func detectFormat(data []byte) string {

	text := strings.ToLower(string(data[:min(len(data), 256)]))

	if strings.HasPrefix(strings.TrimSpace(text), "webvtt") {

		return "vtt"

	}

	return "srt"

}

func normalizeSubtitleUTF8(data []byte) []byte {

	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	if utf8.Valid(data) {

		return data

	}

	runes := make([]rune, len(data))

	for i, b := range data {

		runes[i] = rune(b)

	}

	return []byte(string(runes))

}

type zipSubtitleCandidate struct {

	Name    string
	Payload []byte
	Format  string

}

func extractFromZip(data []byte, season, episode int) ([]byte, string, error) {

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))

	if err != nil {

		return nil, "", err

	}

	var episodeMatches []zipSubtitleCandidate

	var fallback []zipSubtitleCandidate

	for _, file := range reader.File {

		ext := strings.ToLower(filepath.Ext(file.Name))

		format := ""

		switch ext {

		case ".srt":

			format = "srt"

		case ".vtt":

			format = "vtt"

		case ".ass", ".ssa":

			format = strings.TrimPrefix(ext, ".")

		default:

			continue

		}

		if !looksEnglishName(file.Name) {

			continue

		}

		opened, err := file.Open()

		if err != nil {

			continue

		}

		payload, err := io.ReadAll(opened)

		opened.Close()

		if err != nil || len(payload) == 0 {

			continue

		}

		candidate := zipSubtitleCandidate{Name: file.Name, Payload: payload, Format: format}

		if season > 0 && episode > 0 && nameMatchesEpisode(file.Name, season, episode) {

			episodeMatches = append(episodeMatches, candidate)

			continue

		}

		if episode > 0 && nameMatchesEpisode(file.Name, 0, episode) {

			episodeMatches = append(episodeMatches, candidate)

			continue

		}

		fallback = append(fallback, candidate)

	}

	if payload, format := pickZipSubtitleCandidate(episodeMatches); payload != nil {

		return payload, format, nil

	}

	if payload, format := pickZipSubtitleCandidate(fallback); payload != nil {

		return payload, format, nil

	}

	return nil, "", ErrNoSubtitle

}

func pickZipSubtitleCandidate(candidates []zipSubtitleCandidate) ([]byte, string) {

	for _, candidate := range candidates {

		if looksEnglishSubtitle(candidate.Payload) {

			return candidate.Payload, candidate.Format

		}

	}

	return nil, ""

}

func looksLikeSubtitle(data []byte) bool {

	text := strings.ToLower(string(data[:min(len(data), 512)]))

	return strings.Contains(text, "-->") || strings.HasPrefix(strings.TrimSpace(text), "webvtt") || strings.HasPrefix(strings.TrimSpace(text), "[script info]")

}

func min(a, b int) int {

	if a < b {

		return a

	}

	return b

}
