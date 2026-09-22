package sports

import (
	"strings"
	"time"

	catalogindex "streamly/internal/features/catalog/index"
	"streamly/internal/models"
)

func indexSportsMatches(matches []catalogindex.SportsMatchDTO) map[string]catalogindex.SportsMatchDTO {

	byID := make(map[string]catalogindex.SportsMatchDTO, len(matches))

	for _, match := range matches {

		byID[match.ID] = match

	}

	return byID

}

func shouldRefreshSports(pending []models.SportsAlert, byID map[string]catalogindex.SportsMatchDTO, now time.Time) bool {

	deadline := now.Add(sportsAlertRefreshLead)

	for _, alert := range pending {

		match, ok := byID[alert.MatchID]

		if !ok {

			continue

		}

		if match.Delayed {

			return true

		}

		if match.Live || strings.EqualFold(match.Status, "in") {

			return true

		}

		if match.StartsAt > 0 && !time.Unix(match.StartsAt, 0).After(deadline) {

			return true

		}

	}

	return false

}

func decideSportsAlert(match catalogindex.SportsMatchDTO, found bool, now time.Time) sportsAlertDecision {

	if !found {

		return sportsAlertDrop

	}

	if strings.EqualFold(strings.TrimSpace(match.Status), "post") {

		return sportsAlertDrop

	}

	if sportsAlertCanceled(match.StatusDetail) {

		return sportsAlertDrop

	}

	// Rain delay / weather hold: keep the default scheduled-minute path intact
	// by waiting until ESPN clears the delay, then fire as usual.
	if match.Delayed {

		return sportsAlertWait

	}

	if match.StartsAt <= 0 {

		return sportsAlertWait

	}

	start := time.Unix(match.StartsAt, 0).In(now.Location()).Truncate(time.Minute)

	if !now.Before(start) {

		return sportsAlertFire

	}

	return sportsAlertWait

}

func sportsAlertCanceled(detail string) bool {

	blob := strings.ToLower(detail)

	return strings.Contains(blob, "postpon") || strings.Contains(blob, "cancel")

}

func normalizeTeamKey(name string) string {

	return strings.ToLower(strings.TrimSpace(name))

}

func canonicalTeam(name string, matches []catalogindex.SportsMatchDTO) (display, key string, ok bool) {

	key = normalizeTeamKey(name)

	if key == "" {

		return "", "", false

	}

	display = strings.TrimSpace(name)

	for _, match := range matches {

		for _, team := range teamNames(match) {

			if normalizeTeamKey(team) == key {

				return team, key, true

			}

		}

	}

	return display, key, true

}

func teamNames(match catalogindex.SportsMatchDTO) []string {

	names := make([]string, 0, 4)

	add := func(value string) {

		value = strings.TrimSpace(value)

		if value == "" {

			return

		}

		names = append(names, value)

	}

	add(match.HomeTeam)
	add(match.AwayTeam)
	add(match.HomeShortName)
	add(match.AwayShortName)

	if match.HomeTeam == "" && match.AwayTeam == "" {

		for _, team := range titleTeams(match.Title) {

			add(team)

		}

	}

	return names

}

func titleTeams(title string) []string {

	title = strings.TrimSpace(title)
	lower := strings.ToLower(title)

	for _, sep := range []string{" vs. ", " vs ", " at "} {

		i := strings.Index(lower, sep)

		if i < 0 {

			continue

		}

		left := strings.TrimSpace(title[:i])
		right := strings.TrimSpace(title[i+len(sep):])

		if left != "" && right != "" {

			return []string{left, right}

		}

	}

	return nil

}

func matchInvolvesTeam(match catalogindex.SportsMatchDTO, teamKey string) bool {

	if teamKey == "" {

		return false

	}

	for _, name := range teamNames(match) {

		if normalizeTeamKey(name) == teamKey {

			return true

		}

	}

	return false

}

func teamLogosFromMatches(matches []catalogindex.SportsMatchDTO) map[string]string {

	logos := map[string]string{}

	for _, match := range matches {

		if match.HomeTeam != "" && match.HomeLogo != "" {

			logos[normalizeTeamKey(match.HomeTeam)] = match.HomeLogo

		}

		if match.AwayTeam != "" && match.AwayLogo != "" {

			logos[normalizeTeamKey(match.AwayTeam)] = match.AwayLogo

		}

		if match.HomeShortName != "" && match.HomeLogo != "" {

			if _, ok := logos[normalizeTeamKey(match.HomeShortName)]; !ok {

				logos[normalizeTeamKey(match.HomeShortName)] = match.HomeLogo

			}

		}

		if match.AwayShortName != "" && match.AwayLogo != "" {

			if _, ok := logos[normalizeTeamKey(match.AwayShortName)]; !ok {

				logos[normalizeTeamKey(match.AwayShortName)] = match.AwayLogo

			}

		}

	}

	return logos

}
