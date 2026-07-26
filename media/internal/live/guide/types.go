package guide

import (
	"time"

	"mediakit/internal/live/catalog"
)

// Program is a single airing slot on a channel.
type Program struct {

	Title string
	EpisodeTitle string
	Summary string

	StartsAt time.Time
	Runtime int // minutes

	Image string
	Season int
	Episode int

	Genres []string
	Rating string
	Network string

}

// Entry pairs a catalog channel with current and upcoming programs.
type Entry struct {

	Channel catalog.Channel
	Current *Program
	Next *Program

	// Day is a short window of upcoming programs for denser guide UIs.
	Upcoming []Program

}
