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
	ProxyLiveStreams bool `bson:"proxyLiveStreams" json:"proxyLiveStreams"`
	DetectLiveAds bool `bson:"detectLiveAds" json:"detectLiveAds"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`

}

type WatchHistoryItem struct {

	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID  primitive.ObjectID `bson:"userId" json:"userId"`
	Kind    string             `bson:"kind" json:"kind"`
	MediaID int                `bson:"mediaId" json:"mediaId"`
	Title   string             `bson:"title" json:"title"`
	Poster  string             `bson:"poster" json:"poster"`

	Season       int    `bson:"season,omitempty" json:"season,omitempty"`
	Episode      int    `bson:"episode,omitempty" json:"episode,omitempty"`
	EpisodeTitle string `bson:"episodeTitle,omitempty" json:"episodeTitle,omitempty"`
	ChannelID    string `bson:"channelId,omitempty" json:"channelId,omitempty"`

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

type UserProfile struct {

	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID  primitive.ObjectID `bson:"userId" json:"userId"`

	DisplayName string `bson:"displayName" json:"displayName"`
	Bio string `bson:"bio" json:"bio"`
	AccentColor string `bson:"accentColor" json:"accentColor"`
	HistoryVisible bool `bson:"historyVisible" json:"historyVisible"`
	DiscoverVisible *bool `bson:"discoverVisible,omitempty" json:"discoverVisible,omitempty"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`

}

type FriendRequest struct {

	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FromID primitive.ObjectID `bson:"fromId" json:"fromId"`
	ToID   primitive.ObjectID `bson:"toId" json:"toId"`

	Status string `bson:"status" json:"status"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`

}

type PushSubscription struct {

	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"userId" json:"userId"`

	Endpoint string `bson:"endpoint" json:"endpoint"`
	P256dh string `bson:"p256dh" json:"p256dh"`
	Auth string `bson:"auth" json:"auth"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`

}

type SportsAlert struct {

	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"userId" json:"userId"`

	MatchID string `bson:"matchId" json:"matchId"`
	Title string `bson:"title" json:"title"`

	// FromTeam is the normalized team key when this row was created by a team follow.
	FromTeam string `bson:"fromTeam,omitempty" json:"fromTeam,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	FiredAt *time.Time `bson:"firedAt,omitempty" json:"firedAt,omitempty"`

}

type SportsTeamAlert struct {

	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"userId" json:"userId"`

	Team string `bson:"team" json:"team"`
	TeamKey string `bson:"teamKey" json:"teamKey"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`

}
