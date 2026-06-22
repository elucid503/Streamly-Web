package services

import (
	"context"
	"errors"
	"time"

	"streamly/internal/database"
	"streamly/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SettingsService struct {

	db *database.DB

}

func NewSettingsService(db *database.DB) *SettingsService {

	return &SettingsService{db: db}

}

func (s *SettingsService) Get(ctx context.Context, userID string) (*models.UserSettings, error) {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	var settings models.UserSettings

	err = s.db.Settings().FindOne(ctx, bson.M{"userId": oid}).Decode(&settings)

	if errors.Is(err, mongo.ErrNoDocuments) {

		now := time.Now()

		settings = models.UserSettings{

			UserID:           oid,
			PreferredHeight:  1080,
			AutoPlayNext:     true,
			SkipIntro:        true,
			AmbienceEnabled:  true,
			DisablePauseOverlay: false,
			SubtitlesEnabled: true,
			UpdatedAt:        now,

		}

		res, insertErr := s.db.Settings().InsertOne(ctx, settings)

		if insertErr != nil {

			return nil, insertErr

		}

		settings.ID = res.InsertedID.(primitive.ObjectID)

		return &settings, nil

	}

	if err != nil {

		return nil, err

	}

	return &settings, nil

}

type SettingsUpdate struct {

	PreferredHeight  *int  `json:"preferredHeight"`
	AutoPlayNext     *bool `json:"autoPlayNext"`
	SkipIntro        *bool `json:"skipIntro"`
	AmbienceEnabled  *bool `json:"ambienceEnabled"`
	DisablePauseOverlay *bool `json:"disablePauseOverlay"`
	SubtitlesEnabled *bool `json:"subtitlesEnabled"`

}

func (s *SettingsService) Update(ctx context.Context, userID string, update SettingsUpdate) (*models.UserSettings, error) {

	settings, err := s.Get(ctx, userID)

	if err != nil {

		return nil, err

	}

	set := bson.M{"updatedAt": time.Now()}

	if update.PreferredHeight != nil {

		set["preferredHeight"] = *update.PreferredHeight

		settings.PreferredHeight = *update.PreferredHeight

	}

	if update.AutoPlayNext != nil {

		set["autoPlayNext"] = *update.AutoPlayNext

		settings.AutoPlayNext = *update.AutoPlayNext

	}

	if update.SkipIntro != nil {

		set["skipIntro"] = *update.SkipIntro

		settings.SkipIntro = *update.SkipIntro

	}

	if update.AmbienceEnabled != nil {

		set["ambienceEnabled"] = *update.AmbienceEnabled

		settings.AmbienceEnabled = *update.AmbienceEnabled

	}

	if update.SubtitlesEnabled != nil {

		set["subtitlesEnabled"] = *update.SubtitlesEnabled

		settings.SubtitlesEnabled = *update.SubtitlesEnabled

	}

	if update.DisablePauseOverlay != nil {

		set["disablePauseOverlay"] = *update.DisablePauseOverlay

		settings.DisablePauseOverlay = *update.DisablePauseOverlay

	}

	_, err = s.db.Settings().UpdateOne(ctx, bson.M{"_id": settings.ID}, bson.M{"$set": set})

	if err != nil {

		return nil, err

	}

	settings.UpdatedAt = time.Now()

	return settings, nil

}
