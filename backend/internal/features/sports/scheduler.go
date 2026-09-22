package sports

import (
	"context"
	"log"
	"net/url"
	"strings"
	"time"

	catalogindex "streamly/internal/features/catalog/index"
	"streamly/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func (s *SportsAlertsService) expandTeamAlerts(ctx context.Context, matches []catalogindex.SportsMatchDTO) {

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

func (s *SportsAlertsService) materializeTeamAlerts(ctx context.Context, teams []models.SportsTeamAlert, matches []catalogindex.SportsMatchDTO) {

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

func (s *SportsAlertsService) fire(ctx context.Context, alert models.SportsAlert, match catalogindex.SportsMatchDTO) error {

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
