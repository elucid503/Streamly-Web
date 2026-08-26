package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DB struct {

	client *mongo.Client
	db *mongo.Database

}

func Connect(uri string) (*DB, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))

	if err != nil {

		return nil, err

	}

	if err := client.Ping(ctx, nil); err != nil {

		return nil, err

	}

	db := client.Database("streamly-web")

	store := &DB{client: client, db: db}

	if err := store.ensureIndexes(ctx); err != nil {

		return nil, err

	}

	return store, nil

}

func (d *DB) Close(ctx context.Context) error {

	return d.client.Disconnect(ctx)

}

func (d *DB) Users() *mongo.Collection {

	return d.db.Collection("users")

}

func (d *DB) AccessCodes() *mongo.Collection {

	return d.db.Collection("access_codes")

}

func (d *DB) Settings() *mongo.Collection {

	return d.db.Collection("settings")

}

func (d *DB) History() *mongo.Collection {

	return d.db.Collection("history")

}

func (d *DB) Favorites() *mongo.Collection {

	return d.db.Collection("favorites")

}

func (d *DB) ServiceInterruption() *mongo.Collection {

	return d.db.Collection("service_interruption")

}

func (d *DB) Profiles() *mongo.Collection {

	return d.db.Collection("profiles")

}

func (d *DB) FriendRequests() *mongo.Collection {

	return d.db.Collection("friend_requests")

}

func (d *DB) PushSubscriptions() *mongo.Collection {

	return d.db.Collection("push_subscriptions")

}

func (d *DB) SportsAlerts() *mongo.Collection {

	return d.db.Collection("sports_alerts")

}

func (d *DB) SportsTeamAlerts() *mongo.Collection {

	return d.db.Collection("sports_team_alerts")

}

func (d *DB) ensureIndexes(ctx context.Context) error {

	indexes := []struct {

		coll *mongo.Collection
		keys bson.D

		uniq bool

	}{

		{d.Users(), bson.D{{Key: "email", Value: 1}}, true},

		{d.AccessCodes(), bson.D{{Key: "code", Value: 1}}, true},

		{d.Settings(), bson.D{{Key: "userId", Value: 1}}, true},

		{d.History(), bson.D{{Key: "userId", Value: 1}, {Key: "kind", Value: 1}, {Key: "mediaId", Value: 1}, {Key: "season", Value: 1}, {Key: "episode", Value: 1}}, false},
		{d.History(), bson.D{{Key: "userId", Value: 1}, {Key: "updatedAt", Value: -1}}, false},

		{d.Favorites(), bson.D{{Key: "userId", Value: 1}, {Key: "kind", Value: 1}, {Key: "mediaId", Value: 1}, {Key: "channelId", Value: 1}}, true},
		{d.Favorites(), bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}, false},

		{d.Profiles(), bson.D{{Key: "userId", Value: 1}}, true},

		{d.FriendRequests(), bson.D{{Key: "fromId", Value: 1}, {Key: "toId", Value: 1}}, true},
		{d.FriendRequests(), bson.D{{Key: "toId", Value: 1}, {Key: "status", Value: 1}}, false},
		{d.FriendRequests(), bson.D{{Key: "fromId", Value: 1}, {Key: "status", Value: 1}}, false},

		{d.PushSubscriptions(), bson.D{{Key: "userId", Value: 1}, {Key: "endpoint", Value: 1}}, true},
		{d.PushSubscriptions(), bson.D{{Key: "endpoint", Value: 1}}, false},

		{d.SportsAlerts(), bson.D{{Key: "userId", Value: 1}, {Key: "matchId", Value: 1}}, true},

		{d.SportsTeamAlerts(), bson.D{{Key: "userId", Value: 1}, {Key: "teamKey", Value: 1}}, true},

	}

	for _, idx := range indexes {

		opts := options.Index()

		if idx.uniq {

			opts.SetUnique(true)

		}

		if _, err := idx.coll.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: idx.keys, Options: opts}); err != nil {

			return err

		}

	}

	ttl := options.Index().SetExpireAfterSeconds(7 * 24 * 60 * 60)

	if _, err := d.SportsAlerts().Indexes().CreateOne(ctx, mongo.IndexModel{

		Keys: bson.D{{Key: "firedAt", Value: 1}},
		Options: ttl,

	}); err != nil {

		return err

	}

	return nil

}
