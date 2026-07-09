package sports

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	espnScoreboardBase = "https://site.api.espn.com/apis/site/v2/sports"
	espnFetchTimeout   = 12 * time.Second

	// Reject ESPN fixtures that start more than this far from the ntv start.
	// Prevents attaching tonight's Mets score to next week's Mets game.
	maxStartDelta = 18 * time.Hour

	// Each side must score at least this many match points.
	minSidePoints = 2

	// Combined side points must clear this bar (both sides at min → 4).
	minTotalPoints = 4
)

// categoryLeagues maps ntv.cx categories to ESPN sport/league path segments.
var categoryLeagues = map[string][]string{

	"baseball":          {"baseball/mlb"},
	"basketball":        {"basketball/nba", "basketball/wnba", "basketball/mens-college-basketball", "basketball/womens-college-basketball"},
	"american-football": {"football/nfl", "football/college-football"},
	"football":          {"soccer/usa.1", "soccer/eng.1", "soccer/esp.1", "soccer/ger.1", "soccer/ita.1", "soccer/fra.1", "soccer/uefa.champions", "soccer/uefa.europa"},
	"hockey":            {"hockey/nhl"},
	"afl":               {"australian-football/afl"},
	"motor-sports":      {"racing/f1"},
	"rugby":             {"rugby/premiership-rugby", "rugby/six-nations"},

}

type espnScoreboard struct {

	Events []espnEvent `json:"events"`

}

type espnEvent struct {

	Name         string            `json:"name"`
	ShortName    string            `json:"shortName"`
	Date         string            `json:"date"`
	Competitions []espnCompetition `json:"competitions"`
	Status       espnStatus        `json:"status"`

}

type espnCompetition struct {

	Competitors []espnCompetitor `json:"competitors"`
	Status      espnStatus       `json:"status"`
	StartDate   string           `json:"startDate"`

}

type espnCompetitor struct {

	HomeAway string   `json:"homeAway"`
	Score    string   `json:"score"`
	Team     espnTeam `json:"team"`

}

type espnTeam struct {

	DisplayName      string `json:"displayName"`
	ShortDisplayName string `json:"shortDisplayName"`
	Name             string `json:"name"`
	Abbreviation     string `json:"abbreviation"`
	Nickname         string `json:"nickname"`
	Location         string `json:"location"`

}

type espnStatus struct {

	Type espnStatusType `json:"type"`

}

type espnStatusType struct {

	State       string `json:"state"`
	Completed   bool   `json:"completed"`
	Detail      string `json:"detail"`
	ShortDetail string `json:"shortDetail"`
	Description string `json:"description"`

}

// Scoreboard states mirrored from ESPN: pre / in / post.
const (
	StatusPre  = "pre"
	StatusIn   = "in"
	StatusPost = "post"
)

type scoreSnapshot struct {

	HomeNames []string
	AwayNames []string

	HomeScore *int
	AwayScore *int

	StartTime    time.Time
	Status       string
	StatusDetail string

}

// enrichScores pulls ESPN scoreboards for the categories present in matches
// and overlays live status + scores onto matching fixtures.
func (c *Client) enrichScores(matches []Match) {

	leagues := leaguesForMatches(matches)

	if len(leagues) == 0 {

		return

	}

	snapshots := c.fetchScoreSnapshots(leagues)

	if len(snapshots) == 0 {

		return

	}

	// One ESPN fixture may only enrich one ntv match (prevents score reuse).
	used := make(map[int]struct{}, len(snapshots))

	for i := range matches {

		applyScoreSnapshot(&matches[i], snapshots, used)

	}

}

func leaguesForMatches(matches []Match) []string {

	seen := make(map[string]struct{})
	var out []string

	for _, m := range matches {

		for _, league := range categoryLeagues[m.Category] {

			if _, ok := seen[league]; ok {

				continue

			}

			seen[league] = struct{}{}
			out = append(out, league)

		}

	}

	return out

}

func (c *Client) fetchScoreSnapshots(leagues []string) []scoreSnapshot {

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		out []scoreSnapshot
		sem = make(chan struct{}, 4)
	)

	for _, league := range leagues {

		wg.Add(1)
		sem <- struct{}{}

		go func(league string) {

			defer wg.Done()
			defer func() { <-sem }()

			snaps, err := c.fetchLeagueScoreboard(league)

			if err != nil {

				return

			}

			mu.Lock()
			out = append(out, snaps...)
			mu.Unlock()

		}(league)

	}

	wg.Wait()

	return out

}

func (c *Client) fetchLeagueScoreboard(league string) ([]scoreSnapshot, error) {

	url := fmt.Sprintf("%s/%s/scoreboard", espnScoreboardBase, league)

	request, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {

		return nil, err

	}

	request.Header.Set("User-Agent", browserUA)
	request.Header.Set("Accept", "application/json")

	client := c.httpClient

	if client == nil {

		client = &http.Client{Timeout: espnFetchTimeout}

	}

	response, err := client.Do(request)

	if err != nil {

		return nil, err

	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("espn scoreboard %s: status %d", league, response.StatusCode)

	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))

	if err != nil {

		return nil, err

	}

	var board espnScoreboard

	if err := json.Unmarshal(body, &board); err != nil {

		return nil, err

	}

	out := make([]scoreSnapshot, 0, len(board.Events))

	for _, event := range board.Events {

		if snap, ok := scoreSnapshotFromEvent(event); ok {

			out = append(out, snap)

		}

	}

	return out, nil

}

func scoreSnapshotFromEvent(event espnEvent) (scoreSnapshot, bool) {

	if len(event.Competitions) == 0 {

		return scoreSnapshot{}, false

	}

	comp := event.Competitions[0]
	status := comp.Status

	if status.Type.State == "" {

		status = event.Status

	}

	var home, away *espnCompetitor

	for i := range comp.Competitors {

		competitor := &comp.Competitors[i]

		switch competitor.HomeAway {

		case "home":

			home = competitor

		case "away":

			away = competitor

		}

	}

	if home == nil || away == nil {

		return scoreSnapshot{}, false

	}

	detail := status.Type.ShortDetail

	if detail == "" {

		detail = status.Type.Detail

	}

	state := strings.ToLower(strings.TrimSpace(status.Type.State))

	if status.Type.Completed && state != StatusIn {

		state = StatusPost

	}

	if state != StatusPre && state != StatusIn && state != StatusPost {

		state = StatusPre

	}

	startRaw := comp.StartDate

	if startRaw == "" {

		startRaw = event.Date

	}

	startTime, _ := time.Parse(time.RFC3339, startRaw)

	snap := scoreSnapshot{

		HomeNames:    teamNameAliases(home.Team),
		AwayNames:    teamNameAliases(away.Team),
		HomeScore:    parseScore(home.Score),
		AwayScore:    parseScore(away.Score),
		StartTime:    startTime,
		Status:       state,
		StatusDetail: detail,

	}

	return snap, len(snap.HomeNames) > 0 && len(snap.AwayNames) > 0

}

func teamNameAliases(team espnTeam) []string {

	candidates := []string{
		team.DisplayName,
		team.Name,
		team.ShortDisplayName,
		team.Nickname,
		team.Location,
		team.Abbreviation,
	}

	// "Location Nickname" often matches ntv city+mascot forms.
	if team.Location != "" && team.Nickname != "" {

		candidates = append(candidates, team.Location+" "+team.Nickname)

	}

	seen := make(map[string]struct{})
	out := make([]string, 0, len(candidates))

	for _, candidate := range candidates {

		trimmed := strings.TrimSpace(candidate)

		if trimmed == "" {

			continue

		}

		key := normalizeTeamName(trimmed)

		if key == "" {

			continue

		}

		if _, ok := seen[key]; ok {

			continue

		}

		seen[key] = struct{}{}
		out = append(out, trimmed)

	}

	return out

}

func parseScore(raw string) *int {

	raw = strings.TrimSpace(raw)

	if raw == "" {

		return nil

	}

	value, err := strconv.Atoi(raw)

	if err != nil {

		return nil

	}

	return &value

}

func applyScoreSnapshot(match *Match, snapshots []scoreSnapshot, used map[int]struct{}) {

	homeName := ""
	awayName := ""

	if match.HomeTeam != nil {

		homeName = match.HomeTeam.Name

	}

	if match.AwayTeam != nil {

		awayName = match.AwayTeam.Name

	}

	// Prefer structured teams; only fall back to title when both missing.
	if homeName == "" || awayName == "" {

		titleHome, titleAway := splitTitleTeams(match.Title)

		if homeName == "" {

			homeName = titleHome

		}

		if awayName == "" {

			awayName = titleAway

		}

	}

	if homeName == "" || awayName == "" {

		return

	}

	bestIdx := -1
	bestPoints := 0
	bestSwapped := false
	secondPoints := 0

	for i := range snapshots {

		if _, taken := used[i]; taken {

			continue

		}

		snap := &snapshots[i]

		if !startTimesCompatible(match.StartTime, snap.StartTime) {

			continue

		}

		points, swapped, ok := pairMatchPoints(homeName, awayName, snap.HomeNames, snap.AwayNames)

		if !ok {

			continue

		}

		if points > bestPoints {

			secondPoints = bestPoints
			bestPoints = points
			bestIdx = i
			bestSwapped = swapped

			continue

		}

		if points > secondPoints {

			secondPoints = points

		}

	}

	// Ambiguous: two ESPN fixtures score equally well for this match.
	if bestIdx < 0 || bestPoints < minTotalPoints || bestPoints == secondPoints {

		return

	}

	best := &snapshots[bestIdx]
	used[bestIdx] = struct{}{}

	match.StatusDetail = best.StatusDetail
	match.Status = best.Status

	switch best.Status {

	case StatusIn:

		match.Live = true
		assignScores(match, best, bestSwapped)

	case StatusPost:

		match.Live = false
		assignScores(match, best, bestSwapped)

	case StatusPre:

		// Scheduled — keep times/status but don't attach 0–0 placeholder scores.
		match.Live = false
		match.HomeScore = nil
		match.AwayScore = nil

	}

}

func assignScores(match *Match, best *scoreSnapshot, swapped bool) {

	if swapped {

		match.HomeScore = best.AwayScore
		match.AwayScore = best.HomeScore
		return

	}

	match.HomeScore = best.HomeScore
	match.AwayScore = best.AwayScore

}

func startTimesCompatible(matchStart, espnStart time.Time) bool {

	if matchStart.IsZero() || espnStart.IsZero() {

		// Without a comparable clock, fall back to name-only matching.
		return true

	}

	delta := matchStart.Sub(espnStart)

	if delta < 0 {

		delta = -delta

	}

	return delta <= maxStartDelta

}

func splitTitleTeams(title string) (home, away string) {

	lower := strings.ToLower(title)

	for _, sep := range []string{" vs. ", " vs ", " v ", " @ ", " at "} {

		if idx := strings.Index(lower, sep); idx >= 0 {

			left := strings.TrimSpace(title[:idx])
			right := strings.TrimSpace(title[idx+len(sep):])

			// ntv titles are usually "Away vs Home"; treat left as away.
			return right, left

		}

	}

	return "", ""

}

// pairMatchPoints requires BOTH sides to match solidly so a shared team
// (e.g. Mets) cannot attach another opponent's scoreboard.
func pairMatchPoints(homeA, awayA string, homeB, awayB []string) (points int, swapped bool, ok bool) {

	normalHome := bestTeamPoints(homeA, homeB)
	normalAway := bestTeamPoints(awayA, awayB)
	swapHome := bestTeamPoints(homeA, awayB)
	swapAway := bestTeamPoints(awayA, homeB)

	normalOK := normalHome >= minSidePoints && normalAway >= minSidePoints
	swapOK := swapHome >= minSidePoints && swapAway >= minSidePoints

	normalTotal := normalHome + normalAway
	swapTotal := swapHome + swapAway

	switch {

	case normalOK && swapOK:

		if swapTotal > normalTotal {

			return swapTotal, true, swapTotal >= minTotalPoints

		}

		return normalTotal, false, normalTotal >= minTotalPoints

	case normalOK:

		return normalTotal, false, normalTotal >= minTotalPoints

	case swapOK:

		return swapTotal, true, swapTotal >= minTotalPoints

	default:

		return 0, false, false

	}

}

func bestTeamPoints(name string, aliases []string) int {

	best := 0

	for _, alias := range aliases {

		if pts := teamMatchPoints(name, alias); pts > best {

			best = pts

		}

	}

	return best

}

func teamMatchPoints(a, b string) int {

	na, nb := normalizeTeamName(a), normalizeTeamName(b)

	if na == "" || nb == "" {

		return 0

	}

	if na == nb {

		return 4

	}

	// Substring only when the shorter name is substantial (avoid "sox" traps).
	shorter, longer := na, nb

	if len(shorter) > len(longer) {

		shorter, longer = longer, shorter

	}

	if len(shorter) >= 5 && strings.Contains(longer, shorter) {

		return 3

	}

	tokensA := significantTokens(na)
	tokensB := significantTokens(nb)

	if len(tokensA) == 0 || len(tokensB) == 0 {

		return 0

	}

	overlap := 0

	for _, ta := range tokensA {

		for _, tb := range tokensB {

			if ta == tb {

				overlap++
				break

			}

		}

	}

	if overlap >= 2 {

		return 3

	}

	// Single-token nickname/city hit only if the token is long enough and
	// not a generic multi-team fragment (e.g. "new", "york", "sox").
	if overlap == 1 {

		shared := sharedToken(tokensA, tokensB)

		if len(shared) >= 5 && !genericTeamToken(shared) {

			return 2

		}

	}

	return 0

}

func significantTokens(name string) []string {

	parts := strings.Fields(name)
	out := make([]string, 0, len(parts))

	for _, part := range parts {

		if len(part) < 3 || genericTeamToken(part) {

			continue

		}

		out = append(out, part)

	}

	return out

}

func sharedToken(a, b []string) string {

	for _, ta := range a {

		for _, tb := range b {

			if ta == tb {

				return ta

			}

		}

	}

	return ""

}

func genericTeamToken(token string) bool {

	switch token {

	case "new", "york", "los", "angeles", "san", "saint", "st", "city",
		"united", "town", "real", "sporting", "athletic", "club",
		"sox", "bay", "lake", "north", "south", "east", "west":

		return true

	default:

		return false

	}

}

func normalizeTeamName(name string) string {

	name = strings.ToLower(strings.TrimSpace(name))

	if name == "" {

		return ""

	}

	replacer := strings.NewReplacer(
		".", "",
		",", "",
		"'", "",
		"-", " ",
		"_", " ",
		"&", " and ",
	)

	name = replacer.Replace(name)

	// Drop common noise tokens that hurt matching more than they help.
	noise := map[string]struct{}{
		"fc": {}, "sc": {}, "cf": {}, "afc": {}, "the": {}, "w": {}, "women": {}, "womens": {}, "mens": {}, "men": {},
	}

	parts := strings.Fields(name)
	kept := parts[:0]

	for _, part := range parts {

		if _, drop := noise[part]; drop {

			continue

		}

		kept = append(kept, part)

	}

	return strings.Join(kept, " ")

}
