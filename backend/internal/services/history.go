package services

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"streamly/internal/database"
	"streamly/internal/models"
)

type HistoryService struct {
	db *database.DB
}

func NewHistoryService(db *database.DB) *HistoryService {
	return &HistoryService{db: db}
}

type HistoryUpsert struct {
	Kind       string `json:"kind"`
	MediaID    int    `json:"mediaId"`
	Title      string `json:"title"`
	Poster     string `json:"poster"`
	Season     int    `json:"season"`
	Episode    int    `json:"episode"`
	ChannelID  string `json:"channelId"`
	PositionMs int64  `json:"positionMs"`
	DurationMs int64  `json:"durationMs"`
	Completed  bool   `json:"completed"`
}

func (s *HistoryService) List(ctx context.Context, userID string, limit int, mediaID *int) ([]models.WatchHistoryItem, error) {
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	maxLimit := 100
	if mediaID != nil {
		maxLimit = 500
	}
	if limit <= 0 || limit > maxLimit {
		limit = 50
	}

	filter := bson.M{"userId": oid}
	if mediaID != nil {
		filter["mediaId"] = *mediaID
	}

	cur, err := s.db.History().Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var items []models.WatchHistoryItem
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *HistoryService) Upsert(ctx context.Context, userID string, input HistoryUpsert) (*models.WatchHistoryItem, error) {
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"userId":  oid,
		"kind":    input.Kind,
		"mediaId": input.MediaID,
	}
	if input.Kind == "show" {
		filter["season"] = input.Season
		filter["episode"] = input.Episode
	}
	if input.Kind == "live" {
		filter["channelId"] = input.ChannelID
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"title":      input.Title,
			"poster":     input.Poster,
			"season":     input.Season,
			"episode":    input.Episode,
			"channelId":  input.ChannelID,
			"positionMs": input.PositionMs,
			"durationMs": input.DurationMs,
			"completed":  input.Completed,
			"updatedAt":  now,
		},
		"$setOnInsert": bson.M{
			"userId":  oid,
			"kind":    input.Kind,
			"mediaId": input.MediaID,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var item models.WatchHistoryItem
	err = s.db.History().FindOneAndUpdate(ctx, filter, update, opts).Decode(&item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (s *HistoryService) Delete(ctx context.Context, userID, itemID string) error {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	iid, err := primitive.ObjectIDFromHex(itemID)
	if err != nil {
		return err
	}

	_, err = s.db.History().DeleteOne(ctx, bson.M{"_id": iid, "userId": uid})
	return err
}

func (s *HistoryService) GetProgress(ctx context.Context, userID, kind string, mediaID, season, episode int, channelID string) (*models.WatchHistoryItem, error) {
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"userId": oid, "kind": kind, "mediaId": mediaID}
	if kind == "show" {
		filter["season"] = season
		filter["episode"] = episode
	}
	if kind == "live" {
		filter["channelId"] = channelID
	}

	var item models.WatchHistoryItem
	err = s.db.History().FindOne(ctx, filter).Decode(&item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}