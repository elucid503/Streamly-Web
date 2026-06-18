package fileparser

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"mediakit/internal/febbox"
)

var seasonNumberRE = regexp.MustCompile(`(\d+)`)

var episodeNumberREs = []*regexp.Regexp{

	regexp.MustCompile(`(?i)s(\d{1,2})[ ._-]?e(\d{1,4})`),
	regexp.MustCompile(`(?i)\b(\d{1,2})x(\d{1,4})\b`),
	regexp.MustCompile(`(?i)\bepisode[ ._-]?(\d{1,4})\b`),
	regexp.MustCompile(`(?i)\be(\d{1,4})\b`),

}

// ParsedSeason is a season folder extracted from a Febbox file listing.
type ParsedSeason struct {

	Folder febbox.File
	Number int

	Label string

}

// ParsedEpisode is an episode file extracted from a Febbox file listing.
type ParsedEpisode struct {

	File febbox.File

	Number int

	Season int
	SeasonN int

}

// FilesOnly returns only non-directory entries, sorted by name.
func FilesOnly(entries []febbox.File) []febbox.File {

	var files []febbox.File

	for _, entry := range entries {

		if entry.IsDir == 0 {

			files = append(files, entry)

		}

	}

	sort.Slice(files, func(i, j int) bool {

		return strings.Compare(files[i].FileName, files[j].FileName) < 0

	})

	return files

}

// SeasonsOnly returns only directory entries, sorted by name.
func SeasonsOnly(entries []febbox.File) []febbox.File {

	var seasons []febbox.File

	for _, entry := range entries {

		if entry.IsDir == 1 {

			seasons = append(seasons, entry)

		}

	}

	sort.Slice(seasons, func(i, j int) bool {

		return strings.Compare(seasons[i].FileName, seasons[j].FileName) < 0

	})

	return seasons

}

// ParseSeasons extracts season folders and their numbers from a directory listing.
func ParseSeasons(entries []febbox.File) []ParsedSeason {

	seasons := SeasonsOnly(entries)
	out := make([]ParsedSeason, 0, len(seasons))

	for index, folder := range seasons {

		info := seasonInfo(folder.FileName, index+1)

		out = append(out, ParsedSeason{

			Folder: folder,
			Number: info.Number,
			Label:  info.Label,
		})

	}

	return out

}

func seasonInfo(name string, ordinal int) struct {
	Number int
	Label  string
} {

	if match := seasonNumberRE.FindStringSubmatch(name); len(match) > 1 {

		number, _ := strconv.Atoi(match[1])

		return struct {
			Number int
			Label  string
		}{

			Number: number,
			Label:  fmt.Sprintf("Season %d", number),
		}

	}

	return struct {
		Number int
		Label  string
	}{

		Number: ordinal,
		Label:  fmt.Sprintf("Season %d", ordinal),
	}

}

// ParseEpisodes extracts episode files from a directory listing, using FilePreference
// to pick the best copy when multiple files share the same episode number.
func ParseEpisodes(files []febbox.File, seasonNumber int) []ParsedEpisode {

	byNumber := make(map[int]ParsedEpisode)
	fallback := 0

	for _, file := range files {

		season, number := EpisodeNumbers(file.FileName)

		if number == 0 {

			fallback++
			number = fallback

		}

		if season == 0 {

			season = seasonNumber

		}

		candidate := ParsedEpisode{

			File: file,

			Number: number,

			Season:  season,
			SeasonN: seasonNumber,
		}

		if existing, exists := byNumber[number]; !exists {

			byNumber[number] = candidate

		} else if FilePreference(candidate.File.FileName) > FilePreference(existing.File.FileName) {

			byNumber[number] = candidate

		}

	}

	result := make([]ParsedEpisode, 0, len(byNumber))

	for _, ep := range byNumber {

		result = append(result, ep)

	}

	sort.Slice(result, func(i, j int) bool {

		return result[i].Number < result[j].Number

	})

	return result

}

// EpisodeNumbers parses season and episode numbers from a video filename.
func EpisodeNumbers(name string) (season, episode int) {

	for index, pattern := range episodeNumberREs {

		match := pattern.FindStringSubmatch(name)

		if len(match) < 2 {

			continue

		}

		if index < 2 && len(match) > 2 {

			season, _ = strconv.Atoi(match[1])
			episode, _ = strconv.Atoi(match[2])
			return season, episode

		}

		episode, _ = strconv.Atoi(match[1])
		return 0, episode

	}

	return 0, 0

}

// FilePreference scores a filename to prefer H.264 and Blu-ray copies over HEVC or low-quality sources.
// Used for fallback file selection when subtitles or metadata are needed.
func FilePreference(name string) int {

	lower := strings.ToLower(name)
	score := 0

	if strings.Contains(lower, "x265") || strings.Contains(lower, "hevc") || strings.Contains(lower, "h265") {

		score -= 30 // Penalize HEVC; not widely supported in browsers.

	}

	if strings.Contains(lower, "x264") || strings.Contains(lower, "h264") || strings.Contains(lower, "avc") {

		score += 20 // Favor H.264; widely supported and efficient.

	}

	if strings.Contains(lower, "bluray") || strings.Contains(lower, "blu-ray") {

		score += 10 // Favor Blu-ray; typically higher quality.

	}

	if strings.Contains(lower, "1080") {

		score += 5 // Favor 1080p; common high-quality resolution.

	}

	if strings.Contains(lower, "720") {

		score += 2 // Slightly favor 720p; still good quality.

	}

	if strings.Contains(lower, "rarbg") || strings.Contains(lower, "web-dl") {

		score -= 5 // Penalize releases from certain sources; may be less reliable.

	}

	return score

}

// SourceQualityScore estimates the source resolution of a file from its name.
func SourceQualityScore(name string) int {

	lower := strings.ToLower(name)

	switch {

		case strings.Contains(lower, "2160") || strings.Contains(lower, "4k") || strings.Contains(lower, "uhd"):

			return 2160

		case strings.Contains(lower, "1080"):

			return 1080

		case strings.Contains(lower, "720"):

			return 720

		case strings.Contains(lower, "480"):

			return 480

		case strings.Contains(lower, "360"):

			return 360

	}

	return 0

}

// BestSourceFile returns the file with the highest estimated source resolution.
func BestSourceFile(files []febbox.File) febbox.File {

	best := files[0]
	bestScore := SourceQualityScore(files[0].FileName)

	for _, f := range files[1:] {

		if score := SourceQualityScore(f.FileName); score > bestScore {

			best = f
			bestScore = score

		}

	}

	return best

}

// BestSourceFileAtHeight returns the best file whose name indicates the given source resolution.
func BestSourceFileAtHeight(files []febbox.File, height int) (febbox.File, bool) {

	var best febbox.File

	bestPreference := 0
	found := false

	for _, file := range files {

		if SourceQualityScore(file.FileName) != height {

			continue

		}

		preference := FilePreference(file.FileName)

		if !found || preference > bestPreference {

			best = file
			bestPreference = preference
			found = true

		}

	}

	return best, found

}

// AllEpisodeFiles returns every file in the listing that matches the given episode number, without deduplication.
func AllEpisodeFiles(files []febbox.File, seasonNumber, episodeNumber int) []febbox.File {

	var out []febbox.File
	fallback := 0

	for _, file := range files {

		season, number := EpisodeNumbers(file.FileName)

		if number == 0 {

			fallback++
			number = fallback

		}

		if season == 0 {

			season = seasonNumber

		}

		if number == episodeNumber {

			out = append(out, file)

		}

	}

	return out

}
