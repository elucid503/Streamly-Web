package services

import (
	"context"
	"errors"
	"strconv"
	"time"

	"streamly/internal/database"
	"streamly/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrInvalidFavorite = errors.New("invalid favorite")

type FavoritesService struct {
	db *database.DB
}

func NewFavoritesService(db *database.DB) *FavoritesService {

	return &FavoritesService{db: db}

}

type FavoriteUpsert struct {
	Kind      string `json:"kind"`
	MediaID   int    `json:"mediaId"`
	ChannelID string `json:"channelId"`

	Title    string `json:"title"`
	Poster   string `json:"poster"`
	Year     int    `json:"year"`
	Rating   string `json:"rating"`
	Category string `json:"category"`
}

func (s *FavoritesService) List(ctx context.Context, userID string) ([]models.FavoriteItem, error) {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	cur, err := s.db.Favorites().Find(ctx, bson.M{"userId": oid},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))

	if err != nil {

		return nil, err

	}

	defer cur.Close(ctx)

	var items []models.FavoriteItem

	if err := cur.All(ctx, &items); err != nil {

		return nil, err

	}

	return items, nil

}

func (s *FavoritesService) Upsert(ctx context.Context, userID string, input FavoriteUpsert) (*models.FavoriteItem, error) {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	filter, err := favoriteFilter(oid, input.Kind, input.MediaID, input.ChannelID)

	if err != nil {

		return nil, err

	}

	now := time.Now()

	update := bson.M{

		"$set": bson.M{

			"title":    input.Title,
			"poster":   input.Poster,
			"year":     input.Year,
			"rating":   input.Rating,
			"category": input.Category,
		},

		"$setOnInsert": bson.M{

			"userId":    oid,
			"kind":      input.Kind,
			"mediaId":   input.MediaID,
			"channelId": input.ChannelID,
			"createdAt": now,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var item models.FavoriteItem

	err = s.db.Favorites().FindOneAndUpdate(ctx, filter, update, opts).Decode(&item)

	if err != nil {

		return nil, err

	}

	return &item, nil

}

func (s *FavoritesService) Delete(ctx context.Context, userID, kind, key string) error {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return err

	}

	filter, err := favoriteKeyFilter(oid, kind, key)

	if err != nil {

		return err

	}

	_, err = s.db.Favorites().DeleteOne(ctx, filter)

	return err

}

func favoriteFilter(userID primitive.ObjectID, kind string, mediaID int, channelID string) (bson.M, error) {

	if kind != "movie" && kind != "show" && kind != "live" {

		return nil, ErrInvalidFavorite

	}

	filter := bson.M{

		"userId": userID,
		"kind":   kind,
	}

	if kind == "live" {

		if channelID == "" {

			return nil, ErrInvalidFavorite

		}

		filter["mediaId"] = 0
		filter["channelId"] = channelID

		return filter, nil

	}

	if mediaID <= 0 {

		return nil, ErrInvalidFavorite

	}

	filter["mediaId"] = mediaID
	filter["channelId"] = ""

	return filter, nil

}

func favoriteKeyFilter(userID primitive.ObjectID, kind, key string) (bson.M, error) {

	if kind == "live" {

		return favoriteFilter(userID, kind, 0, key)

	}

	oid, err := strconv.Atoi(key)

	if err != nil {

		return nil, ErrInvalidFavorite

	}

	return favoriteFilter(userID, kind, oid, "")

}
