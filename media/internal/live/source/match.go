package source

import (
	"strings"
	"unicode"
)

// normalizeName strips quality/region noise for fuzzy channel matching.
func normalizeName(name string) string {

	name = strings.ToLower(strings.TrimSpace(name))

	if name == "" {

		return ""

	}

	// Drop parenthetical notes: "ESPN (US)" -> "ESPN"
	if i := strings.Index(name, "("); i >= 0 {

		name = strings.TrimSpace(name[:i])

	}

	replacer := strings.NewReplacer(
		"&", " and ",
		"'", "",
		"’", "",
		".", " ",
		"_", " ",
		"-", " ",
		"/", " ",
	)

	name = replacer.Replace(name)

	// Strip common trailing tokens.
	for {

		trimmed := false

		for _, suffix := range []string{
			" usa", " us", " uk", " ca", " canada", " hd", " fhd", " uhd", " 4k",
			" network", " channel", " tv", " television",
		} {

			if !strings.HasSuffix(name, suffix) || len(name) <= len(suffix)+1 {

				continue

			}

			leftover := strings.TrimSpace(name[:len(name)-len(suffix)])

			// Keep "YES Network" / "USA Network" intact — stripping to "yes"
			// or "usa" collides with unrelated short tokens.
			if isBrandSuffix(suffix) && len(leftover) < 4 {

				continue

			}

			name = leftover
			trimmed = true

		}

		if !trimmed {

			break

		}

	}

	var b strings.Builder
	prevSpace := false

	for _, r := range name {

		if unicode.IsLetter(r) || unicode.IsDigit(r) {

			b.WriteRune(r)
			prevSpace = false

		} else if !prevSpace {

			b.WriteByte(' ')
			prevSpace = true

		}

	}

	return strings.Join(strings.Fields(b.String()), " ")

}

func isBrandSuffix(suffix string) bool {

	switch suffix {

	case " network", " channel", " tv", " television":

		return true

	}

	return false

}

// nameCandidates builds match keys for a resolve request.
func nameCandidates(req Request) []string {

	seen := map[string]struct{}{}
	var out []string

	add := func(raw string) {

		n := normalizeName(raw)

		if n == "" {

			return

		}

		if _, ok := seen[n]; ok {

			return

		}

		seen[n] = struct{}{}
		out = append(out, n)

	}

	add(req.Name)
	add(req.Network)

	for _, a := range req.AltNames {

		add(a)

	}

	// Also try without trailing "news"/"sports" word flips for Fox News etc.
	if n := normalizeName(req.Name); strings.HasSuffix(n, " news") {

		add(strings.TrimSpace(strings.TrimSuffix(n, " news")) + " news")

	}

	return out

}

// matchScore rates how well a provider channel name matches the request.
// Higher is better; 0 means no match.
func matchScore(req Request, providerName string) int {

	target := normalizeName(providerName)

	if target == "" {

		return 0

	}

	best := 0

	for _, cand := range nameCandidates(req) {

		if cand == target {

			if best < 100 {

				best = 100

			}

			continue

		}

		// Provider often has "espn usa" while catalog has "espn".
		if strings.HasPrefix(target, cand+" ") || strings.HasPrefix(cand, target+" ") {

			if best < 90 {

				best = 90

			}

			continue

		}

		if strings.Contains(target, cand) && len(cand) >= 4 {

			if best < 70 {

				best = 70

			}

			continue

		}

		if strings.Contains(cand, target) && len(target) >= 4 {

			if best < 60 {

				best = 60

			}

		}

	}

	return best

}

// bestMatch returns the best-scoring name from candidates above minScore.
func bestMatch(req Request, names []string, minScore int) (string, int) {

	bestName := ""
	best := 0

	for _, name := range names {

		score := matchScore(req, name)

		if score > best {

			best = score
			bestName = name

		}

	}

	if best < minScore {

		return "", 0

	}

	return bestName, best

}
