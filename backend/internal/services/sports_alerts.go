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

const (

	sportsAlertTick = 30 * time.Second
	kickoffLead = 90 * time.Second
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

func (s *SportsAlertsService) List(ctx context.Context, userID string) ([]SportsAlertDTO, error) {

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

	out := make([]SportsAlertDTO, 0, len(rows))

	for _, row := range rows {

		out = append(out, SportsAlertDTO{MatchID: row.MatchID, Title: row.Title})

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

	matches, err := s.matches.LiveSports()

	if err != nil {

		log.Printf("sports-alerts: matches: %v", err)
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

func (s *SportsAlertsService) fire(ctx context.Context, alert models.SportsAlert, match catalog.SportsMatchDTO) error {

	channelName := match.Channel.Name
	body := "Starting now"

	if channelName != "" {

		body = "Starting now · Watch on " + channelName

	}

	err := s.push.SendToUser(ctx, alert.UserID, SportsPushPayload{

		Title: match.Title,
		Body: body,
		URL: "/live/" + url.PathEscape(match.Channel.ID),
		Tag: "sports-" + match.ID,

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

	if match.Channel == nil || strings.TrimSpace(match.Channel.ID) == "" {

		return sportsAlertWait

	}

	if match.Live || strings.EqualFold(strings.TrimSpace(match.Status), "in") {

		return sportsAlertFire

	}

	if match.StartsAt <= 0 {

		return sportsAlertWait

	}

	start := time.Unix(match.StartsAt, 0)

	if !start.After(now.Add(kickoffLead)) {

		return sportsAlertFire

	}

	return sportsAlertWait

}
