package catalog

import "strings"

// teamSportsExact are regional / team-specific sports networks whose iptv-org
// records are often missing category, owner, or network metadata (YES Network).
var teamSportsExact = map[string]bool{

	"yes network": true,
	"yes 2": true,
	"yes2": true,
	"yes2 overflow": true,
	"sny": true,
	"nesn": true,
	"nesn plus": true,
	"masn": true,
	"masn2": true,
	"msg": true,
	"msg 2": true,
	"msg plus": true,
	"msg plus 2": true,
	"msg western new york": true,
	"msg sportsnet": true,
	"chsn": true,
	"chicago sports network": true,
	"space city home network": true,
	"marquee sports network": true,
	"marquee": true,
	"monumental sports network": true,
	"sportsnet new york": true,
	"sportsnet pittsburgh": true,
	"altitude sports": true,
	"altitude 2": true,
	"root sports northwest": true,
	"tva sports": true,
	"tva sports 2": true,
	"tva sports 3": true,
	"tsn": true,
	"tsn1": true,
	"tsn2": true,
	"tsn3": true,
	"tsn4": true,
	"tsn5": true,
	"tsn 4k": true,
	"rds": true,

}

// teamSportsPrefixes match entire RSN families (FanDuel Sports Network Detroit,
// Spectrum SportsNet LA, NBC Sports Bay Area, …).
var teamSportsPrefixes = []string{

	"fanduel sports network",
	"spectrum sportsnet",
	"nbc sports",
	"root sports",
	"sportsnet",
	"altitude",
	"msg",
	"nesn",
	"masn",
	"tsn",
	"tva sports",

}

// isTeamSportsChannel reports whether this is a regional / team RSN we should
// keep even when iptv-org leaves categories or owners empty.
func isTeamSportsChannel(raw iptvChannel) bool {

	name := normalizeTeamKey(raw.Name)

	if name == "" {

		return false

	}

	if teamSportsExact[name] {

		return true

	}

	for _, prefix := range teamSportsPrefixes {

		if name == prefix || strings.HasPrefix(name, prefix+" ") {

			return true

		}

	}

	return false

}

func normalizeTeamKey(s string) string {

	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "’", "")
	s = strings.Join(strings.Fields(s), " ")

	return s

}

// extraTeamAltNames adds ESPN-style short labels so sports matching can hit
// "YES" / "SNY" against catalog names like "Yes Network".
func extraTeamAltNames(name string) []string {

	switch normalizeTeamKey(name) {

	case "yes network":

		return []string{"YES"}

	case "yes 2", "yes2":

		return []string{"YES 2", "YES2"}

	case "yes2 overflow":

		return []string{"YES 2", "YES2"}

	case "sportsnet new york":

		return []string{"SNY"}

	case "chicago sports network":

		return []string{"CHSN"}

	case "spectrum sportsnet la":

		return []string{"SportsNet LA", "Sportsnet LA"}

	}

	return nil

}

func mergeAltNames(existing, extra []string, primary string) []string {

	seen := map[string]bool{}
	out := make([]string, 0, len(existing)+len(extra))

	add := func(s string) {

		s = strings.TrimSpace(s)

		if s == "" {

			return

		}

		key := strings.ToLower(s)

		if seen[key] || key == strings.ToLower(strings.TrimSpace(primary)) {

			return

		}

		seen[key] = true
		out = append(out, s)

	}

	for _, s := range existing {

		add(s)

	}

	for _, s := range extra {

		add(s)

	}

	return out

}
