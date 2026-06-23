package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`

	IsAdmin bool `bson:"isAdmin" json:"isAdmin"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type AccessCode struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code string             `bson:"code" json:"code"`

	CreatedBy primitive.ObjectID `bson:"createdBy" json:"createdBy"`

	MaxUses int `bson:"maxUses" json:"maxUses"`
	Uses    int `bson:"uses" json:"uses"`

	ExpiresAt *time.Time `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
}

type UserSettings struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"userId" json:"userId"`

	PreferredHeight  int  `bson:"preferredHeight" json:"preferredHeight"`
	AutoPlayNext     bool `bson:"autoPlayNext" json:"autoPlayNext"`
	SkipIntro        bool `bson:"skipIntro" json:"skipIntro"`
	DisablePauseOverlay bool `bson:"disablePauseOverlay" json:"disablePauseOverlay"`
	AmbienceEnabled  bool `bson:"ambienceEnabled" json:"ambienceEnabled"`
	SubtitlesEnabled bool `bson:"subtitlesEnabled" json:"subtitlesEnabled"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type WatchHistoryItem struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID  primitive.ObjectID `bson:"userId" json:"userId"`
	Kind    string             `bson:"kind" json:"kind"`
	MediaID int                `bson:"mediaId" json:"mediaId"`
	Title   string             `bson:"title" json:"title"`
	Poster  string             `bson:"poster" json:"poster"`

	Season    int    `bson:"season,omitempty" json:"season,omitempty"`
	Episode   int    `bson:"episode,omitempty" json:"episode,omitempty"`
	ChannelID string `bson:"channelId,omitempty" json:"channelId,omitempty"`

	PositionMs int64 `bson:"positionMs" json:"positionMs"`
	DurationMs int64 `bson:"durationMs" json:"durationMs"`
	Completed  bool  `bson:"completed" json:"completed"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type ServiceInterruption struct {

	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Enabled bool               `bson:"enabled" json:"enabled"`
	Title   string             `bson:"title" json:"title"`
	Message string             `bson:"message" json:"message"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`

}

type FavoriteItem struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Kind      string             `bson:"kind" json:"kind"`
	MediaID   int                `bson:"mediaId" json:"mediaId"`
	ChannelID string             `bson:"channelId,omitempty" json:"channelId,omitempty"`

	Title    string `bson:"title" json:"title"`
	Poster   string `bson:"poster" json:"poster"`
	Year     int    `bson:"year,omitempty" json:"year,omitempty"`
	Rating   string `bson:"rating,omitempty" json:"rating,omitempty"`
	Category string `bson:"category,omitempty" json:"category,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}
