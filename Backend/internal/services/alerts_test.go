package services

import (
	"testing"
	"time"

	"streamly/internal/services/catalog"
)

func TestDecideSportsAlertFiresAtScheduledMinute(t *testing.T) {

	start := time.Date(2026, 8, 26, 19, 5, 30, 0, time.Local)
	match := catalog.SportsMatchDTO{

		ID: "espn-1",
		Title: "Yankees at Red Sox",
		Status: "pre",
		StartsAt: start.Unix(),

	}

	minute := start.Truncate(time.Minute)

	if got := decideSportsAlert(match, true, minute.Add(-time.Second)); got != sportsAlertWait {

		t.Fatalf("before minute: got %v want wait", got)

	}

	if got := decideSportsAlert(match, true, minute); got != sportsAlertFire {

		t.Fatalf("at minute: got %v want fire", got)

	}

}

func TestDecideSportsAlertIgnoresLiveAndChannel(t *testing.T) {

	start := time.Date(2026, 8, 26, 19, 5, 0, 0, time.Local)
	match := catalog.SportsMatchDTO{

		ID: "espn-1",
		Status: "in",
		Live: true,
		StartsAt: start.Unix(),

	}

	if got := decideSportsAlert(match, true, start.Add(-time.Minute)); got != sportsAlertWait {

		t.Fatalf("live before schedule: got %v want wait", got)

	}

	if got := decideSportsAlert(match, true, start); got != sportsAlertFire {

		t.Fatalf("live at schedule without channel: got %v want fire", got)

	}

}

func TestDecideSportsAlertWaitsOnRainDelay(t *testing.T) {

	start := time.Date(2026, 8, 26, 19, 5, 0, 0, time.Local)
	match := catalog.SportsMatchDTO{

		ID: "espn-1",
		Status: "pre",
		Delayed: true,
		StartsAt: start.Unix(),
		StatusDetail: "Rain Delay",

	}

	if got := decideSportsAlert(match, true, start.Add(15*time.Minute)); got != sportsAlertWait {

		t.Fatalf("delayed past start: got %v want wait", got)

	}

	match.Delayed = false
	match.StatusDetail = "Top 1st"

	if got := decideSportsAlert(match, true, start.Add(15*time.Minute)); got != sportsAlertFire {

		t.Fatalf("delay lifted: got %v want fire", got)

	}

}

func TestDecideSportsAlertDropCanceled(t *testing.T) {

	match := catalog.SportsMatchDTO{

		ID: "espn-1",
		Status: "pre",
		StartsAt: time.Now().Unix(),
		StatusDetail: "Postponed",

	}

	if got := decideSportsAlert(match, true, time.Now()); got != sportsAlertDrop {

		t.Fatalf("postponed: got %v want drop", got)

	}

}

func TestMatchInvolvesTeam(t *testing.T) {

	match := catalog.SportsMatchDTO{

		Title: "New York Yankees at Boston Red Sox",
		HomeTeam: "Boston Red Sox",
		AwayTeam: "New York Yankees",
		HomeShortName: "Red Sox",
		AwayShortName: "Yankees",

	}

	if !matchInvolvesTeam(match, "boston red sox") {

		t.Fatal("expected home team match")

	}

	if !matchInvolvesTeam(match, "yankees") {

		t.Fatal("expected away short name match")

	}

	if matchInvolvesTeam(match, "dodgers") {

		t.Fatal("did not expect unrelated team")

	}

}
