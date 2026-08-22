package sports

import (
	"strings"

	"mediakit/internal/live/catalog"
)

// broadcastCandidate is a live TV/stream outlet from ESPN for one event.
type broadcastCandidate struct {

	Name string
	// Kind is "tv", "streaming", "radio", or "".
	Kind string
	// Market is "national", "home", "away", or "".
	Market string
	// Prefer is a ranking boost (higher = try first).
	Prefer int

}

// skipBroadcasts are OTT-only outlets we do not map to linear catalog channels
// (unless nothing else is available — then they contribute no search terms).
var skipBroadcasts = map[string]bool{

	"peacock": true,
	"netflix": true,
	"apple tv": true,
	"apple tv+": true,
	"amazon": true,
	"amazon prime": true,
	"prime video": true,
	"max": true,
	"hulu": true,
	"youtube": true,
	"youtube tv": true,
	"paramount+": true,
	"paramount plus": true,
	"disney+": true,
	"fubo": true,
	"mlb.tv": true,
	"mlbtv": true,
	"nba league pass": true,
	"nhl.tv": true,
	"nhltv": true,
	"espn+": true,
	"espn unlmtd": true,
	"espn unlimited": true,
	"radio": true,
	"siriusxm": true,
	"eradm": true,

}

// broadcastAliases maps ESPN short names → catalog search labels (ordered).
// Keys are normalized (lowercase, collapsed spaces).
var broadcastAliases = map[string][]string{

	"sny": {"SportsNet New York", "SNY"},
	"sportsnet new york": {"SportsNet New York", "SNY"},
	"yes": {"YES Network", "YES"},
	"yes network": {"YES Network", "YES"},
	"yes 2": {"YES 2", "YES Network", "YES"},
	"yes2": {"YES 2", "YES Network", "YES"},
	"wpix": {"PIX 11", "PIX11", "WPIX", "CW PIX 11"},
	"pix 11": {"PIX 11", "PIX11", "WPIX"},
	"pix11": {"PIX 11", "PIX11", "WPIX"},
	"cw pix 11": {"PIX 11", "PIX11", "WPIX"},
	"nesn": {"NESN"},
	"nesn plus": {"NESN Plus", "NESN"},
	"masn": {"MASN"},
	"masn2": {"MASN"},
	"marquee sports net": {"Marquee Sports Network", "Marquee"},
	"marquee": {"Marquee Sports Network"},
	"sportsnet la": {"Spectrum SportsNet LA", "SportsNet LA"},
	"spectrum sportsnet la": {"Spectrum SportsNet LA"},
	"spectrum sportsnet": {"Spectrum SportsNet", "Spectrum SportsNet LA"},
	"spectrum sports net": {"Spectrum SportsNet", "Spectrum SportsNet LA"},
	"nbc sports ba": {"NBC Sports Bay Area"},
	"nbc sports bay area": {"NBC Sports Bay Area"},
	"nbc sports ca": {"NBC Sports California"},
	"nbc sports california": {"NBC Sports California"},
	// Canadian Sportsnet (distinct from SNY / SportsNet LA).
	"sportsnet": {"Sportsnet One", "Sportsnet 360"},
	"nbc sports boston": {"NBC Sports Boston"},
	"nbc sports chicago": {"NBC Sports Chicago"},
	"nbc sports now": {"NBC Sports NOW"},
	"space city home network": {"Space City Home Network", "FanDuel Sports Network Southwest Houston"},
	"chsn": {"Chicago Sports Network", "NBC Sports Chicago"},
	"chicago sports network": {"Chicago Sports Network", "NBC Sports Chicago"},
	"rangers sports network": {"FanDuel Sports Network Southwest", "Rangers Sports Network"},
	"sportsnet/tva": {"Sportsnet One", "Sportsnet 360", "TVA Sports"},
	"sportsnet one": {"Sportsnet One"},
	"sportsnet 360": {"Sportsnet 360"},
	"tva": {"TVA Sports"},
	"tva sports": {"TVA Sports"},
	"msg": {"MSG"},
	"msg 2": {"MSG 2", "MSG"},
	"msg plus": {"MSG Plus"},
	"msg western new york": {"MSG Western New York", "MSG"},
	"altitude": {"Altitude Sports"},
	"altitude sports": {"Altitude Sports"},
	"root sports northwest": {"Root Sports Northwest"},
	"root sports": {"Root Sports Northwest"},
	"bravesvision": {"FanDuel Sports Network Southeast", "BravesVision"},
	"bally sports": {"FanDuel Sports Network"},

	// National linear.
	"espn": {"ESPN"},
	"espn2": {"ESPN2", "ESPN 2"},
	"espnu": {"ESPNU"},
	"espn news": {"ESPNews"},
	"abc": {"ABC"},
	"nbc": {"NBC"},
	"cbs": {"CBS"},
	"fox": {"Fox", "FOX"},
	"fs1": {"Fox Sports 1"},
	"fs2": {"Fox Sports 2"},
	"fox sports 1": {"Fox Sports 1"},
	"fox sports 2": {"Fox Sports 2"},
	"tnt": {"TNT"},
	"tbs": {"TBS"},
	"usa": {"USA Network"},
	"usa network": {"USA Network"},
	"nfl network": {"NFL Network"},
	"nfl redzone": {"NFL RedZone", "NFL Network"},
	"mlb network": {"MLB Network"},
	"nba tv": {"NBA TV"},
	"nhl network": {"NHL Network"},
	"golf channel": {"Golf Channel"},
	"cbs sports network": {"CBS Sports Network"},
	"bein sports": {"beIN Sports"},
	"nbc/peacock": {"NBC", "Peacock"},
	"espn/abc": {"ESPN", "ABC"},
	"espn unlmtd/mlb.tv": {"ESPN", "MLB Network"},

}

// teamRSNs maps team name fragments → preferred regional sports networks.
// Used when ESPN only lists OTT (MLB.TV, Peacock) or as home/away enrichment.
var teamRSNs = map[string][]string{

	// MLB
	"yankees": {"YES Network", "YES"},
	"mets": {"SportsNet New York", "SNY"},
	"red sox": {"NESN"},
	"dodgers": {"Spectrum SportsNet LA", "SportsNet LA"},
	"cubs": {"Marquee Sports Network"},
	"white sox": {"Chicago Sports Network", "NBC Sports Chicago", "CHSN"},
	"giants": {"NBC Sports Bay Area"},
	"athletics": {"NBC Sports California"},
	"angels": {"FanDuel Sports Network West"},
	"padres": {"FanDuel Sports Network", "Padres.TV"},
	"mariners": {"Root Sports Northwest"},
	// Texas Rangers vs NY Rangers: see teamRSNsByCategory.
	"astros": {"Space City Home Network", "FanDuel Sports Network Southwest Houston"},
	"phillies": {"NBC Sports Philadelphia", "NBC Sports"},
	"braves": {"FanDuel Sports Network Southeast"},
	"orioles": {"MASN"},
	"nationals": {"MASN"},
	"blue jays": {"Sportsnet", "Sportsnet One"},
	"guardians": {"FanDuel Sports Network Great Lakes", "FanDuel Sports Network Ohio Cleveland"},
	"tigers": {"FanDuel Sports Network Detroit"},
	"royals": {"FanDuel Sports Network Kansas City"},
	"twins": {"FanDuel Sports Network North", "Twins.TV"},
	"brewers": {"FanDuel Sports Network Wisconsin"},
	"cardinals": {"FanDuel Sports Network Midwest"},
	"reds": {"FanDuel Sports Network Ohio Cincinnati", "FanDuel Sports Network Ohio"},
	"pirates": {"SportsNet Pittsburgh", "AT&T SportsNet Pittsburgh"},
	"rockies": {"Altitude Sports"},
	"diamondbacks": {"FanDuel Sports Network Arizona"},
	"marlins": {"FanDuel Sports Network Florida"},
	"rays": {"FanDuel Sports Network Sun"},

	// NBA (common RSNs / nationals still handled via ESPN broadcasts)
	"knicks": {"MSG"},
	"nets": {"YES Network", "YES"},
	"lakers": {"Spectrum SportsNet", "Spectrum SportsNet LA"},
	"clippers": {"FanDuel Sports Network SoCal", "FanDuel Sports Network West"},
	"celtics": {"NBC Sports Boston", "NESN"},
	"warriors": {"NBC Sports Bay Area"},
	"bulls": {"Chicago Sports Network", "NBC Sports Chicago"},
	"heat": {"FanDuel Sports Network Sun", "FanDuel Sports Network Florida"},
	"bucks": {"FanDuel Sports Network Wisconsin"},
	"mavericks": {"FanDuel Sports Network Southwest"},
	"rockets": {"Space City Home Network", "FanDuel Sports Network Southwest Houston"},
	"spurs": {"FanDuel Sports Network Southwest San Antonio"},
	"suns": {"FanDuel Sports Network Arizona"},
	"nuggets": {"Altitude Sports"},
	"timberwolves": {"FanDuel Sports Network North"},
	"thunder": {"FanDuel Sports Network Oklahoma"},
	"jazz": {"Jazz+", "KJZZ"},
	"kings": {"NBC Sports California"},
	"blazers": {"Root Sports Northwest"},
	"trail blazers": {"Root Sports Northwest"},
	"pelicans": {"FanDuel Sports Network New Orleans", "Gulf Coast Sports"},
	"grizzlies": {"FanDuel Sports Network Southeast"},
	"hornets": {"FanDuel Sports Network Southeast"},
	"hawks": {"FanDuel Sports Network Southeast"},
	"magic": {"FanDuel Sports Network Florida"},
	"wizards": {"Monumental Sports Network"},
	"pistons": {"FanDuel Sports Network Detroit"},
	"pacers": {"FanDuel Sports Network Indiana"},
	"cavaliers": {"FanDuel Sports Network Ohio Cleveland"},
	"cavs": {"FanDuel Sports Network Ohio Cleveland"},
	"raptors": {"Sportsnet", "TSN"},
	"76ers": {"NBC Sports Philadelphia"},
	"sixers": {"NBC Sports Philadelphia"},

	// NHL (NY Rangers handled in teamRSNsByCategory)
	"islanders": {"MSG", "MSG Plus"},
	"devils": {"MSG", "MSG Sportsnet"},
	"bruins": {"NESN"},
	"maple leafs": {"Sportsnet", "TSN"},
	"canadiens": {"TSN", "RDS"},
	"penguins": {"SportsNet Pittsburgh"},
	"capitals": {"Monumental Sports Network"},
	"flyers": {"NBC Sports Philadelphia"},
	"blackhawks": {"Chicago Sports Network", "NBC Sports Chicago"},
	"red wings": {"FanDuel Sports Network Detroit"},
	"blues": {"FanDuel Sports Network Midwest"},
	"avalanche": {"Altitude Sports"},
	"stars": {"FanDuel Sports Network Southwest"},
	"predators": {"FanDuel Sports Network South"},
	"hurricanes": {"FanDuel Sports Network South"},
	"panthers": {"FanDuel Sports Network Florida"},
	"lightning": {"FanDuel Sports Network Sun"},
	"wild": {"FanDuel Sports Network North"},
	"jets": {"TSN"},
	"oilers": {"Sportsnet"},
	"flames": {"Sportsnet"},
	"canucks": {"Sportsnet"},
	"kraken": {"Root Sports Northwest"},
	"sharks": {"NBC Sports California"},
	"ducks": {"Victory+", "FanDuel Sports Network SoCal"},
	// LA Kings vs Sacramento Kings: teamRSNsByCategory
	"golden knights": {"SCRIPPS", "KMCC"},

	// NFL prefers national — team RSNs less common for primetime
}

// nfl/mlb disambiguation for "rangers" / "kings"
var teamRSNsByCategory = map[string]map[string][]string{

	"baseball": {
		"rangers": {"FanDuel Sports Network Southwest", "Rangers Sports Network"},
	},
	"hockey": {
		"rangers": {"MSG"},
		"kings": {"FanDuel Sports Network SoCal", "FanDuel Sports Network West"},
	},
	"basketball": {
		"kings": {"NBC Sports California"},
		"rangers": {}, // N/A
	},

}

// categoryDefaultNetworks is the last-resort national lineup.
var categoryDefaultNetworks = map[string][]string{

	"baseball": {"MLB Network", "ESPN", "TBS", "Fox", "FOX"},
	"basketball": {"NBA TV", "ESPN", "TNT", "ABC"},
	"american-football": {"NFL Network", "ESPN", "Fox", "FOX", "CBS", "NBC"},
	"hockey": {"NHL Network", "ESPN", "TNT"},
	"football": {"ESPN", "Fox Sports 1", "FS1", "beIN Sports"},
	"motor-sports": {"ESPN", "ABC", "Fox Sports 1"},
	"golf": {"Golf Channel", "CBS", "NBC"},
	"mma": {"ESPN", "UFC Fight Pass"},

}

// matchChannel picks the best catalog channel for a fixture using live ESPN
// broadcast data first, then team RSN maps, then national defaults.
func matchChannel(m Match, cat *catalog.Catalog) *MatchedChannel {

	if ch := matchBroadcastChannel(m, cat); ch != nil {

		return ch

	}

	return matchTeamOrDefaultChannel(m, cat)

}

func matchBroadcastChannel(m Match, cat *catalog.Catalog) *MatchedChannel {

	if cat == nil {

		return nil

	}

	for _, term := range expandBroadcastLabels(m.Broadcasts) {

		if ch := findCatalogChannel(cat, term); ch != nil {

			return ch

		}

	}

	return nil

}

func matchTeamOrDefaultChannel(m Match, cat *catalog.Catalog) *MatchedChannel {

	if cat == nil {

		return nil

	}

	for _, term := range teamSearchTerms(m) {

		if ch := findCatalogChannel(cat, term); ch != nil {

			return ch

		}

	}

	for _, name := range categoryDefaultNetworks[m.Category] {

		if ch := findCatalogChannel(cat, name); ch != nil {

			return ch

		}

	}

	return nil

}

func hasOTTBroadcast(names []string) bool {

	for _, name := range names {

		for _, part := range splitBroadcastName(name) {

			if skipBroadcasts[normalizeKey(part)] {

				return true

			}

		}

	}

	return false

}

// True when every named outlet is OTT/radio (or there are no names at all).
func broadcastsAreOTTOnly(names []string) bool {

	if len(names) == 0 {

		return true

	}

	sawOutlet := false

	for _, name := range names {

		for _, part := range splitBroadcastName(name) {

			key := normalizeKey(part)

			if key == "" {

				continue

			}

			sawOutlet = true

			if !skipBroadcasts[key] {

				return false

			}

		}

	}

	return sawOutlet || len(names) == 0

}

// expandBroadcastLabels turns ESPN outlet names into catalog lookup labels.
func expandBroadcastLabels(names []string) []string {

	var terms []string
	seen := map[string]bool{}

	add := func(label string) {

		label = strings.TrimSpace(label)

		if label == "" {

			return

		}

		key := normalizeKey(label)

		if seen[key] || skipBroadcasts[key] {

			return

		}

		seen[key] = true
		terms = append(terms, label)

	}

	for _, name := range names {

		key := normalizeKey(name)

		if key == "" {

			continue

		}

		// Expand compounds first (ESPN/ABC, Sportsnet/TVA).
		parts := splitBroadcastName(name)

		if len(parts) == 0 {

			parts = []string{name}

		}

		for _, part := range parts {

			pk := normalizeKey(part)

			if pk == "" || skipBroadcasts[pk] {

				continue

			}

			if aliases, ok := broadcastAliases[pk]; ok {

				for _, a := range aliases {

					add(a)

				}

				continue

			}

			add(part)

		}

		// Full-string alias (e.g. "nbc sports ba").
		if aliases, ok := broadcastAliases[key]; ok {

			for _, a := range aliases {

				add(a)

			}

		}

	}

	return terms

}

func teamSearchTerms(m Match) []string {

	var terms []string
	seen := map[string]bool{}

	addAll := func(labels []string) {

		for _, l := range labels {

			k := normalizeKey(l)

			if l == "" || seen[k] {

				continue

			}

			seen[k] = true
			terms = append(terms, l)

		}

	}

	// Home first — local RSN is usually the intended watch path.
	if m.HomeTeam != nil {

		addAll(rsnForTeam(m.Category, m.HomeTeam.Name))

	}

	if m.AwayTeam != nil {

		addAll(rsnForTeam(m.Category, m.AwayTeam.Name))

	}

	return terms

}

func rsnForTeam(category, teamName string) []string {

	n := normalizeKey(teamName)

	if n == "" {

		return nil

	}

	if byCat := teamRSNsByCategory[category]; byCat != nil {

		for key, labels := range byCat {

			if strings.Contains(n, key) && len(labels) > 0 {

				return labels

			}

		}

	}

	// Longest key wins to prefer "trail blazers" over "blazers" order — iterate all.
	var bestKey string
	var best []string

	for key, labels := range teamRSNs {

		if labels == nil || len(labels) == 0 {

			continue

		}

		if strings.Contains(n, key) && len(key) > len(bestKey) {

			bestKey = key
			best = labels

		}

	}

	return best

}

func findCatalogChannel(cat *catalog.Catalog, name string) *MatchedChannel {

	name = strings.TrimSpace(name)

	if name == "" {

		return nil

	}

	if ch, ok := cat.FindByExactName(name); ok {

		return &MatchedChannel{ChannelID: ch.ID, Name: ch.Name, Logo: ch.Logo}

	}

	// Prefer exact-ish hits from search.
	hits := cat.Search(name, 12)

	norm := normalizeKey(name)
	var best *catalog.Channel
	bestScore := 0

	for i := range hits {

		ch := &hits[i]
		score := scoreChannelName(norm, ch)

		if score > bestScore {

			bestScore = score
			best = ch

		}

	}

	// High bar avoids "Rangers" → random AFN Sports style false positives.
	if best == nil || bestScore < 70 {

		return nil

	}

	return &MatchedChannel{ChannelID: best.ID, Name: best.Name, Logo: best.Logo}

}

func scoreChannelName(norm string, ch *catalog.Channel) int {

	best := 0

	for _, key := range channelNormKeys(ch) {

		score := nameSimilarity(norm, key)

		if score > best {

			best = score

		}

	}

	if best == 0 {

		return 0

	}

	// Prefer sports-category channels when ambiguous (e.g. TBS vs local news).
	if strings.EqualFold(ch.Category, "Sports") {

		best += 8

	}

	// Prefer US feeds for US leagues.
	if strings.EqualFold(ch.Country.Code, "us") {

		best += 3

	}

	return best

}

func channelNormKeys(ch *catalog.Channel) []string {

	seen := map[string]bool{}
	keys := make([]string, 0, 1+len(ch.AltNames))

	add := func(s string) {

		k := normalizeKey(s)

		if k == "" || seen[k] {

			return

		}

		seen[k] = true
		keys = append(keys, k)

	}

	add(ch.Name)

	for _, alt := range ch.AltNames {

		add(alt)

	}

	return keys

}

func nameSimilarity(norm, cn string) int {

	if cn == norm {

		return 100

	}

	qWords := len(strings.Fields(norm))
	nWords := len(strings.Fields(cn))

	if strings.HasPrefix(cn, norm) && len(norm) >= 4 {

		// Avoid "Sportsnet" matching "SportsNet New York".
		if nWords > qWords+1 {

			return 40

		}

		return 85

	}

	if strings.HasPrefix(norm, cn) && len(cn) >= 5 {

		if qWords > nWords+1 {

			return 40

		}

		return 75

	}

	if strings.Contains(cn, norm) && len(norm) >= 5 {

		if nWords > qWords+1 {

			return 40

		}

		return 65

	}

	if strings.Contains(norm, cn) && len(cn) >= 6 {

		return 55

	}

	return 0

}

func splitBroadcastName(name string) []string {

	name = strings.TrimSpace(name)

	if name == "" {

		return nil

	}

	// Only split compound listings ("ESPN/ABC", "Sportsnet|TVA") — never on
	// spaces, so "NBC Sports CA" and "Sportsnet LA" stay intact for alias maps.
	for _, sep := range []string{"/", "|"} {

		if !strings.Contains(name, sep) {

			continue

		}

		var out []string

		for _, piece := range strings.Split(name, sep) {

			piece = strings.TrimSpace(piece)

			if piece != "" {

				out = append(out, piece)

			}

		}

		if len(out) > 0 {

			return out

		}

	}

	return []string{name}

}

func normalizeKey(s string) string {

	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "’", "")
	s = strings.Join(strings.Fields(s), " ")

	return s

}

// primaryBroadcastLabel picks a human-facing network label for the UI.
func primaryBroadcastLabel(broadcasts []broadcastCandidate) string {

	names := make([]string, 0, len(broadcasts))

	for _, b := range broadcasts {

		if strings.TrimSpace(b.Name) != "" {

			names = append(names, b.Name)

		}

	}

	terms := expandBroadcastLabels(names)

	if len(terms) > 0 {

		return terms[0]

	}

	if len(names) > 0 {

		return names[0]

	}

	return ""

}
