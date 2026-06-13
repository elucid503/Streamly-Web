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
	db     *mongo.Database
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

func (d *DB) ProxyTokens() *mongo.Collection {
	return d.db.Collection("proxy_tokens")
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
		{d.ProxyTokens(), bson.D{{Key: "token", Value: 1}}, true},
		{d.ProxyTokens(), bson.D{{Key: "expiresAt", Value: 1}}, false},
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

	_, _ = d.ProxyTokens().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})

	return nil
}
