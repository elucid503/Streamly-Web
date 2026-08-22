package sports

import "strings"

// ESPN OTT labels → NTV search terms. Keys match normalizeKey.
var ottCatalogTerms = map[string][]string{

	"apple tv": {"Apple TV", "AppleTV", "Apple TV+"},
	"apple tv+": {"Apple TV", "AppleTV", "Apple TV+"},
	"espn+": {"ESPN+", "ESPN PLUS", "ESPN Plus"},
	"espn unlmtd": {"ESPN+", "ESPN PLUS", "ESPN Plus"},
	"espn unlimited": {"ESPN+", "ESPN PLUS", "ESPN Plus"},
	"peacock": {"Peacock"},
	"amazon": {"Amazon Prime", "Prime Video"},
	"amazon prime": {"Amazon Prime", "Prime Video"},
	"prime video": {"Amazon Prime", "Prime Video"},
	"paramount+": {"Paramount+", "Paramount Plus"},
	"paramount plus": {"Paramount+", "Paramount Plus"},
	"mlbtv": {"MLB.TV", "MLB TV"},
	"nba league pass": {"NBA League Pass", "League Pass"},
	"nhltv": {"NHL.TV", "NHL TV"},
	"max": {"Max", "HBO Max"},
	"netflix": {"Netflix"},
	"disney+": {"Disney+", "Disney Plus"},
	"hulu": {"Hulu"},
	"fubo": {"Fubo", "Fubo Sports"},
	"youtube": {"YouTube"},
	"youtube tv": {"YouTube TV"},

}

func ottSearchTerms(broadcasts []string) []string {

	var terms []string
	seen := map[string]bool{}

	add := func(label string) {

		label = strings.TrimSpace(label)

		if label == "" {

			return

		}

		key := normalizeKey(label)

		if seen[key] {

			return

		}

		seen[key] = true
		terms = append(terms, label)

	}

	for _, name := range broadcasts {

		for _, part := range splitBroadcastName(name) {

			pk := normalizeKey(part)

			if labels, ok := ottCatalogTerms[pk]; ok {

				for _, label := range labels {

					add(label)

				}

				continue

			}

			if skipBroadcasts[pk] {

				add(part)

			}

		}

	}

	return terms

}
