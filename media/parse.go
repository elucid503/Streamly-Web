package mediakit

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"mediakit/internal/febbox"
	"mediakit/internal/showbox"
)

var seasonNumberRE = regexp.MustCompile(`(\d+)`)

var episodeNumberREs = []*regexp.Regexp{

	regexp.MustCompile(`(?i)s(\d{1,2})[ ._-]?e(\d{1,4})`),
	regexp.MustCompile(`(?i)\b(\d{1,2})x(\d{1,4})\b`),
	regexp.MustCompile(`(?i)\bepisode[ ._-]?(\d{1,4})\b`),
	regexp.MustCompile(`(?i)\be(\d{1,4})\b`),

}

func parseTitleDetails(raw map[string]any) TitleDetails {

	text := func(key string) string {

		value, ok := raw[key]

		if !ok || value == nil || value == "" {

			return ""

		}

		return showbox.DecodeText(fmt.Sprint(value))

	}

	return TitleDetails{

		Title: fallback(text("title"), "Unknown title"),
		Year: text("year"),

		Poster: fallback(text("poster"), fallback(text("poster_org"), text("poster_min"))),
		Banner: fallback(text("banner"), fallback(text("backdrop"), fallback(text("cover"), text("still")))),

		Description: text("description"),

		IMDBRating: text("imdb_rating"),

		TMDBId: intFromAny(raw["tmdb_id"]),
		IMDBId: text("imdb_id"),

		EpisodeTitles: episodeTitleMap(raw),

	}

}

func episodeTitleMap(raw map[string]any) map[string]string {

	episodes, ok := raw["episode"].([]any)

	if !ok {

		return nil

	}

	titles := make(map[string]string)

	for _, item := range episodes {

		data, ok := item.(map[string]any)

		if !ok {

			continue

		}

		season, _ := data["season"].(float64)
		number, _ := data["episode"].(float64)

		title := showbox.DecodeText(fmt.Sprint(data["title"]))

		if season > 0 && number > 0 && title != "" {

			titles[fmt.Sprintf("%d:%d", int(season), int(number))] = title

		}

	}

	if len(titles) == 0 {

		return nil

	}

	return titles

}

func fallback(values ...string) string {

	for _, value := range values {

		if value != "" {

			return value

		}

	}

	return ""

}

func intFromAny(value any) int {

	switch typed := value.(type) {

		case int:

			return typed

		case int64:

			return int(typed)

		case float64:

			return int(typed)

		case string:

			parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
			return parsed

		default:

			return 0

	}

}

func filesOnly(entries []febbox.File) []febbox.File {

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

func seasonsOnly(entries []febbox.File) []febbox.File {

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

type parsedSeason struct {

	Folder febbox.File
	Number int

	Label string

}

func parseSeasons(entries []febbox.File) []parsedSeason {

	seasons := seasonsOnly(entries)
	out := make([]parsedSeason, 0, len(seasons))

	for index, folder := range seasons {

		info := seasonInfo(folder.FileName, index+1)

		out = append(out, parsedSeason{

			Folder: folder,
			Number: info.Number,
			Label: info.Label,

		})

	}

	return out

}

func seasonInfo(name string, ordinal int) struct {

	Number int
	Label string

} {

	if match := seasonNumberRE.FindStringSubmatch(name); len(match) > 1 {

		number, _ := strconv.Atoi(match[1])

		return struct {

			Number int
			Label string

		}{
			Number: number,
		 	Label: fmt.Sprintf("Season %d", number),

		}

	}

	return struct {

		Number int
		Label string

	}{

		Number: ordinal,
		Label: fmt.Sprintf("Season %d", ordinal),

	}

}

type parsedEpisode struct {

	File febbox.File

	Number int

	Season int
	SeasonN int

}

func parseEpisodes(files []febbox.File, seasonNumber int) []parsedEpisode {

	byNumber := make(map[int]parsedEpisode)
	fallback := 0

	for _, file := range files {

		season, number := episodeNumbers(file.FileName)

		if number == 0 {

			fallback++
			number = fallback

		}

		if season == 0 {

			season = seasonNumber

		}

		candidate := parsedEpisode{

			File: file,

			Number: number,

			Season: season,
			SeasonN: seasonNumber,

		}

		if existing, exists := byNumber[number]; !exists {

			byNumber[number] = candidate

		} else if filePreference(candidate.File.FileName) > filePreference(existing.File.FileName) {

			byNumber[number] = candidate

		}

	}

	result := make([]parsedEpisode, 0, len(byNumber))

	for _, ep := range byNumber {

		result = append(result, ep)

	}

	sort.Slice(result, func(i, j int) bool {

		return result[i].Number < result[j].Number

	})

	return result

}

func episodeNumbers(name string) (season, episode int) {

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

func hitFromResult(result showbox.SearchResult) SearchHit {

	return SearchHit{

		ID: result.ID,
		Kind: MediaKind(result.BoxType),

		Title: result.Title,
		Year: result.Year,
		Poster: result.Poster,

		Description: result.Description,

		IMDBRating: result.IMDBRating,

	}

}
