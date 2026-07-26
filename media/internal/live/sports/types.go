package sports

import "time"

// Team is one side of a sports match.
type Team struct {

	Name string
	Logo string
	Abbreviation string

}

// MatchedChannel is an optional catalog channel suggested for watching.
// Linked from live ESPN broadcast data and/or team RSN maps.
type MatchedChannel struct {

	ChannelID string
	Name string
	Logo string

}

// Match is a sports fixture from scoreboard sources (not stream providers).
type Match struct {

	ID string
	Title string
	Category string
	League string

	StartTime time.Time
	Live bool

	HomeTeam *Team
	AwayTeam *Team

	HomeScore *int
	AwayScore *int
	StatusDetail string
	// Status is lifecycle state: pre / in / post.
	Status string

	// Broadcasts are live TV/stream outlets from ESPN for this event
	// (e.g. "SNY", "ESPN", "NBC"), ordered by preference.
	Broadcasts []string

	// Broadcast is the primary outlet label for display.
	Broadcast string

	Channel *MatchedChannel

}

// Scoreboard states.
const (
	StatusPre = "pre"
	StatusIn = "in"
	StatusPost = "post"
)
