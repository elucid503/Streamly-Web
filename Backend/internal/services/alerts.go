package services

import (
	"context"
	"errors"
	"log"
	"net/url"
	"strings"
	"time"

	"streamly/internal/database"
	"streamly/internal/models"
	"streamly/internal/services/catalog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrSportsAlertMatch = errors.New("match not found")
var ErrSportsAlertTeam = errors.New("team required")

const (

	sportsAlertTick = 30 * time.Second
	sportsAlertRefreshLead = 3 * time.Minute

)

type sportsAlertDecision int

const (

	sportsAlertWait sportsAlertDecision = iota
	sportsAlertFire
	sportsAlertDrop

)

type SportsMatchLookup interface {

	LiveSports() ([]catalog.SportsMatchDTO, error)
	RefreshLiveSports() []catalog.SportsMatchDTO

}

type SportsAlertsService struct {

	db *database.DB
	push *PushService
	matches SportsMatchLookup

	cancel context.CancelFunc

}

func NewSportsAlertsService(db *database.DB, push *PushService, matches SportsMatchLookup) *SportsAlertsService {

	return &SportsAlertsService{

		db: db,
		push: push,
		matches: matches,

	}

}

type SportsAlertDTO struct {

	MatchID string `json:"matchId"`
	Title string `json:"title"`

}

type SportsTeamAlertDTO struct {

	Team string `json:"team"`
	Logo string `json:"logo,omitempty"`

}

type SportsAlertsListDTO struct {

	Matches []SportsAlertDTO `json:"matches"`
	Teams []SportsTeamAlertDTO `json:"teams"`

}

func (s *SportsAlertsService) List(ctx context.Context, userID string) (*SportsAlertsListDTO, error) {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	cur, err := s.db.SportsAlerts().Find(ctx, bson.M{

		"userId": oid,
		"firedAt": bson.M{"$exists": false},

	}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))

	if err != nil {

		return nil, err

	}

	defer cur.Close(ctx)

	var rows []models.SportsAlert

	if err := cur.All(ctx, &rows); err != nil {

		return nil, err

	}

	teamCur, err := s.db.SportsTeamAlerts().Find(ctx, bson.M{

		"userId": oid,

	}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))

	if err != nil {

		return nil, err

	}

	defer teamCur.Close(ctx)

	var teamRows []models.SportsTeamAlert

	if err := teamCur.All(ctx, &teamRows); err != nil {

		return nil, err

	}

	matches, _ := s.matches.LiveSports()
	logos := teamLogosFromMatches(matches)

	out := &SportsAlertsListDTO{

		Matches: make([]SportsAlertDTO, 0, len(rows)),
		Teams: make([]SportsTeamAlertDTO, 0, len(teamRows)),

	}

	for _, row := range rows {

		if strings.TrimSpace(row.FromTeam) != "" {

			continue

		}

		out.Matches = append(out.Matches, SportsAlertDTO{MatchID: row.MatchID, Title: row.Title})

	}

	for _, row := range teamRows {

		out.Teams = append(out.Teams, SportsTeamAlertDTO{

			Team: row.Team,
			Logo: logos[row.TeamKey],

		})

	}

	return out, nil

}

func (s *SportsAlertsService) Subscribe(ctx context.Context, userID, matchID string) (*SportsAlertDTO, error) {

	if !s.push.Configured() {

		return nil, ErrPushNotConfigured

	}

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	matchID = strings.TrimSpace(matchID)

	if matchID == "" {

		return nil, ErrSportsAlertMatch

	}

	match, ok := s.findMatch(matchID)

	if !ok {

		return nil, ErrSportsAlertMatch

	}

	if strings.EqualFold(match.Status, "post") {

		return nil, ErrSportsAlertMatch

	}

	now := time.Now()

	_, err = s.db.SportsAlerts().UpdateOne(ctx, bson.M{

		"userId": oid,
		"matchId": matchID,

	}, bson.M{

		"$set": bson.M{

			"title": match.Title,

		},
		"$setOnInsert": bson.M{

			"userId": oid,
			"matchId": matchID,
			"createdAt": now,

		},
		"$unset": bson.M{

			"firedAt": "",
			"fromTeam": "",

		},

	}, options.Update().SetUpsert(true))

	if err != nil {

		return nil, err

	}

	return &SportsAlertDTO{MatchID: matchID, Title: match.Title}, nil

}

func (s *SportsAlertsService) Unsubscribe(ctx context.Context, userID, matchID string) error {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return err

	}

	matchID = strings.TrimSpace(matchID)

	if matchID == "" {

		return ErrSportsAlertMatch

	}

	_, err = s.db.SportsAlerts().DeleteOne(ctx, bson.M{"userId": oid, "matchId": matchID})

	return err

}

func (s *SportsAlertsService) SubscribeTeam(ctx context.Context, userID, team string) (*SportsTeamAlertDTO, error) {

	if !s.push.Configured() {

		return nil, ErrPushNotConfigured

	}

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	matches, _ := s.matches.LiveSports()
	display, key, ok := canonicalTeam(team, matches)

	if !ok {

		return nil, ErrSportsAlertTeam

	}

	now := time.Now()

	_, err = s.db.SportsTeamAlerts().UpdateOne(ctx, bson.M{

		"userId": oid,
		"teamKey": key,

	}, bson.M{

		"$set": bson.M{

			"team": display,

		},
		"$setOnInsert": bson.M{

			"userId": oid,
			"teamKey": key,
			"createdAt": now,

		},

	}, options.Update().SetUpsert(true))

	if err != nil {

		return nil, err

	}

	s.materializeTeamAlerts(ctx, []models.SportsTeamAlert{{

		UserID: oid,
		Team: display,
		TeamKey: key,

	}}, matches)

	logos := teamLogosFromMatches(matches)

	return &SportsTeamAlertDTO{Team: display, Logo: logos[key]}, nil

}

func (s *SportsAlertsService) UnsubscribeTeam(ctx context.Context, userID, team string) error {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return err

	}

	key := normalizeTeamKey(team)

	if key == "" {

		return ErrSportsAlertTeam

	}

	_, err = s.db.SportsTeamAlerts().DeleteOne(ctx, bson.M{"userId": oid, "teamKey": key})

	if err != nil {

		return err

	}

	_, err = s.db.SportsAlerts().DeleteMany(ctx, bson.M{

		"userId": oid,
		"fromTeam": key,
		"firedAt": bson.M{"$exists": false},

	})

	return err

}

func (s *SportsAlertsService) Start(ctx context.Context) {

	child, cancel := context.WithCancel(ctx)

	s.cancel = cancel

	go func() {

		s.tick(child)

		ticker := time.NewTicker(sportsAlertTick)

		defer ticker.Stop()

		for {

			select {

			case <-child.Done():

				return

			case <-ticker.C:

				s.tick(child)

			}

		}

	}()

}

func (s *SportsAlertsService) Stop() {

	if s.cancel != nil {

		s.cancel()

	}

}

func (s *SportsAlertsService) findMatch(matchID string) (catalog.SportsMatchDTO, bool) {

	matches, err := s.matches.LiveSports()

	if err != nil {

		return catalog.SportsMatchDTO{}, false

	}

	for _, match := range matches {

		if match.ID == matchID {

			return match, true

		}

	}

	return catalog.SportsMatchDTO{}, false

}

func (s *SportsAlertsService) tick(ctx context.Context) {

	if !s.push.Configured() {

		return

	}

	matches, err := s.matches.LiveSports()

	if err != nil {

		log.Printf("sports-alerts: matches: %v", err)
		return

	}

	s.expandTeamAlerts(ctx, matches)

	cur, err := s.db.SportsAlerts().Find(ctx, bson.M{"firedAt": bson.M{"$exists": false}})

	if err != nil {

		log.Printf("sports-alerts: list pending: %v", err)
		return

	}

	var pending []models.SportsAlert

	if err := cur.All(ctx, &pending); err != nil {

		_ = cur.Close(ctx)
		log.Printf("sports-alerts: decode pending: %v", err)
		return

	}

	_ = cur.Close(ctx)

	if len(pending) == 0 {

		return

	}

	byID := indexSportsMatches(matches)

	if shouldRefreshSports(pending, byID, time.Now()) {

		matches = s.matches.RefreshLiveSports()
		byID = indexSportsMatches(matches)

	}

	now := time.Now()

	for _, alert := range pending {

		select {

		case <-ctx.Done():

			return

		default:

		}

		match, ok := byID[alert.MatchID]
		decision := decideSportsAlert(match, ok, now)

		switch decision {

		case sportsAlertDrop:

			_, _ = s.db.SportsAlerts().DeleteOne(ctx, bson.M{"_id": alert.ID})

		case sportsAlertFire:

			if err := s.fire(ctx, alert, match); err != nil {

				log.Printf("sports-alerts: fire %s: %v", alert.MatchID, err)

			}

		}

	}

}

func (s *SportsAlertsService) expandTeamAlerts(ctx context.Context, matches []catalog.SportsMatchDTO) {

	cur, err := s.db.SportsTeamAlerts().Find(ctx, bson.M{})

	if err != nil {

		log.Printf("sports-alerts: list teams: %v", err)
		return

	}

	var teams []models.SportsTeamAlert

	if err := cur.All(ctx, &teams); err != nil {

		_ = cur.Close(ctx)
		log.Printf("sports-alerts: decode teams: %v", err)
		return

	}

	_ = cur.Close(ctx)

	s.materializeTeamAlerts(ctx, teams, matches)

}

func (s *SportsAlertsService) materializeTeamAlerts(ctx context.Context, teams []models.SportsTeamAlert, matches []catalog.SportsMatchDTO) {

	if len(teams) == 0 || len(matches) == 0 {

		return

	}

	now := time.Now()

	for _, team := range teams {

		if team.TeamKey == "" {

			continue

		}

		for _, match := range matches {

			if strings.EqualFold(strings.TrimSpace(match.Status), "post") {

				continue

			}

			if !matchInvolvesTeam(match, team.TeamKey) {

				continue

			}

			_, err := s.db.SportsAlerts().UpdateOne(ctx, bson.M{

				"userId": team.UserID,
				"matchId": match.ID,

			}, bson.M{

				"$setOnInsert": bson.M{

					"userId": team.UserID,
					"matchId": match.ID,
					"title": match.Title,
					"fromTeam": team.TeamKey,
					"createdAt": now,

				},

			}, options.Update().SetUpsert(true))

			if err != nil {

				log.Printf("sports-alerts: materialize %s %s: %v", team.TeamKey, match.ID, err)

			}

		}

	}

}

func (s *SportsAlertsService) fire(ctx context.Context, alert models.SportsAlert, match catalog.SportsMatchDTO) error {

	title := match.Title

	if strings.TrimSpace(title) == "" {

		title = alert.Title

	}

	body := "Starting now"
	target := "/"
	tag := "sports-" + alert.MatchID

	if match.Channel != nil && strings.TrimSpace(match.Channel.ID) != "" {

		target = "/live/" + url.PathEscape(match.Channel.ID)

		if name := strings.TrimSpace(match.Channel.Name); name != "" {

			body = "Starting now · Watch on " + name

		}

	}

	err := s.push.SendToUser(ctx, alert.UserID, SportsPushPayload{

		Title: title,
		Body: body,
		URL: target,
		Tag: tag,

	})

	if err != nil {

		return err

	}

	fired := time.Now()

	_, err = s.db.SportsAlerts().UpdateOne(ctx, bson.M{

		"_id": alert.ID,
		"firedAt": bson.M{"$exists": false},

	}, bson.M{

		"$set": bson.M{"firedAt": fired},

	})

	return err

}

func indexSportsMatches(matches []catalog.SportsMatchDTO) map[string]catalog.SportsMatchDTO {

	byID := make(map[string]catalog.SportsMatchDTO, len(matches))

	for _, match := range matches {

		byID[match.ID] = match

	}

	return byID

}

func shouldRefreshSports(pending []models.SportsAlert, byID map[string]catalog.SportsMatchDTO, now time.Time) bool {

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

func decideSportsAlert(match catalog.SportsMatchDTO, found bool, now time.Time) sportsAlertDecision {

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

func canonicalTeam(name string, matches []catalog.SportsMatchDTO) (display, key string, ok bool) {

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

func teamNames(match catalog.SportsMatchDTO) []string {

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

func matchInvolvesTeam(match catalog.SportsMatchDTO, teamKey string) bool {

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

func teamLogosFromMatches(matches []catalog.SportsMatchDTO) map[string]string {

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
