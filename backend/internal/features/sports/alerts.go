package sports

import (
	"context"
	"errors"
	"strings"
	"time"

	"streamly/internal/database"
	catalogindex "streamly/internal/features/catalog/index"
	"streamly/internal/models"

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

	LiveSports() ([]catalogindex.SportsMatchDTO, error)
	RefreshLiveSports() []catalogindex.SportsMatchDTO

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

func (s *SportsAlertsService) findMatch(matchID string) (catalogindex.SportsMatchDTO, bool) {

	matches, err := s.matches.LiveSports()

	if err != nil {

		return catalogindex.SportsMatchDTO{}, false

	}

	for _, match := range matches {

		if match.ID == matchID {

			return match, true

		}

	}

	return catalogindex.SportsMatchDTO{}, false

}
