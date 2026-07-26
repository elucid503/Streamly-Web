package catalog

import (
	"sort"
	"strings"
)

// popularNames is a curated ranking of major US cable/broadcast channels.
// Used only for ordering; the catalog itself is built from metadata sources.
var popularNames = []string{

	"ESPN", "ESPN2", "ESPNU", "ESPNews", "ESPN Deportes",
	"ABC", "CBS", "NBC", "Fox", "FOX", "The CW", "PBS", "PBS Kids",
	"CNN", "Fox News", "MSNBC", "CNBC", "HLN", "Newsmax", "BBC News",
	"HBO", "HBO2", "Showtime", "Starz", "Cinemax",
	"TNT", "TBS", "USA Network", "FX", "FXX", "AMC", "Syfy", "Paramount Network",
	"Discovery Channel", "History", "National Geographic", "Nat Geo Wild", "TLC", "A&E", "Animal Planet", "Science",
	"Comedy Central", "MTV", "VH1", "BET", "Freeform",
	"Disney Channel", "Disney Junior", "Disney XD", "Nickelodeon", "Nick Jr.", "Cartoon Network", "Adult Swim",
	"Fox Sports 1", "Fox Sports 2", "NBA TV", "NFL Network", "MLB Network", "NHL Network", "Golf Channel", "Tennis Channel",
	"CBS Sports Network", "NBC Sports", "beIN Sports",
	"Bravo", "E!", "Food Network", "HGTV", "Travel Channel", "The Weather Channel", "Lifetime", "Oxygen",
	"Hallmark Channel", "ION", "MeTV", "Bounce TV",
	"Telemundo", "Univision", "UniMás", "Galavisión",

}

func rankPopular(channels []Channel) []Channel {

	rank := make(map[string]int, len(popularNames))

	for i, name := range popularNames {

		rank[strings.ToLower(name)] = i

	}

	ranked := dedupeByName(channels)

	sort.SliceStable(ranked, func(i, j int) bool {

		hi, hj := ranked[i].Enriched && ranked[i].Logo != "", ranked[j].Enriched && ranked[j].Logo != ""

		if hi != hj {

			return hi

		}

		ri, oki := rank[strings.ToLower(ranked[i].Name)]
		rj, okj := rank[strings.ToLower(ranked[j].Name)]

		if oki && okj {

			return ri < rj

		}

		return oki && !okj

	})

	return ranked

}

func dedupeByName(channels []Channel) []Channel {

	bestByName := make(map[string]Channel, len(channels))
	order := make([]string, 0, len(channels))

	for _, ch := range channels {

		key := strings.ToLower(strings.TrimSpace(ch.Name))

		if key == "" {

			continue

		}

		existing, ok := bestByName[key]

		if !ok {

			bestByName[key] = ch
			order = append(order, key)
			continue

		}

		if channelPriority(ch) > channelPriority(existing) {

			bestByName[key] = ch

		}

	}

	out := make([]Channel, 0, len(order))

	for _, key := range order {

		out = append(out, bestByName[key])

	}

	return out

}

func channelPriority(ch Channel) int {

	score := 0

	if ch.Enriched {

		score += 2

	}

	if ch.Logo != "" {

		score++

	}

	if ch.Country.Code == "us" {

		score += 2

	}

	if len(ch.Owners) > 0 {

		score++

	}

	return score

}

func isMajorName(name string) bool {

	n := strings.ToLower(strings.TrimSpace(name))

	for _, major := range popularNames {

		if strings.ToLower(major) == n {

			return true

		}

	}

	return false

}
