package sports

import "time"

// Team is one side of a sports match.
type Team struct {
	Name string
}

// MatchedChannel is the 24/7 channel a match's broadcast was matched to.
type MatchedChannel struct {
	ChannelID string
	Name      string
	Logo      string
}

// Match is a single sports fixture from the ntv.cx match feed.
type Match struct {
	ID       string
	Title    string
	Category string

	StartTime time.Time
	Live      bool

	HomeTeam *Team
	AwayTeam *Team

	// Optional live scoreboard fields (ESPN enrichment).
	HomeScore    *int
	AwayScore    *int
	StatusDetail string
	// Status is ESPN lifecycle state when known: pre / in / post.
	Status string

	Channel *MatchedChannel
}

type rawTeam struct {
	Name  string `json:"name"`
	Badge string `json:"badge"`
}

type rawTeams struct {
	Home *rawTeam `json:"home"`
	Away *rawTeam `json:"away"`
}

type rawSource struct {
	Source string `json:"source"`
	ID     string `json:"id"`
}

type rawMatch struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`

	Date int64 `json:"date"`

	Teams   rawTeams    `json:"teams"`
	Sources []rawSource `json:"sources"`

	Live bool `json:"live"`
}

type getMatchesResponse struct {
	Success bool `json:"success"`

	Live    []rawMatch `json:"live"`
	NonLive []rawMatch `json:"nonLive"`
	All     []rawMatch `json:"all"`

	Categories []string `json:"categories"`
}

func (r rawMatch) toMatch() Match {

	match := Match{

		ID:       r.ID,
		Title:    r.Title,
		Category: r.Category,

		StartTime: time.UnixMilli(r.Date),
		Live:      r.Live,
	}

	if r.Teams.Home != nil && r.Teams.Home.Name != "" {

		match.HomeTeam = &Team{Name: r.Teams.Home.Name}

	}

	if r.Teams.Away != nil && r.Teams.Away.Name != "" {

		match.AwayTeam = &Team{Name: r.Teams.Away.Name}

	}

	return match

}