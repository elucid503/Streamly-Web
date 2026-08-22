package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"streamly/internal/config"
	"streamly/internal/database"
	"streamly/internal/models"

	webpush "github.com/SherClockHolmes/webpush-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrPushNotConfigured = errors.New("push notifications are not configured")
var ErrInvalidPushSubscription = errors.New("invalid push subscription")

type PushService struct {

	db *database.DB
	cfg *config.Config

}

func NewPushService(db *database.DB, cfg *config.Config) *PushService {

	return &PushService{db: db, cfg: cfg}

}

func (s *PushService) Configured() bool {

	if s.cfg == nil {

		return false

	}

	return strings.TrimSpace(s.cfg.VAPIDPublicKey) != "" && strings.TrimSpace(s.cfg.VAPIDPrivateKey) != ""

}

func (s *PushService) PublicKey() string {

	if s.cfg == nil {

		return ""

	}

	return strings.TrimSpace(s.cfg.VAPIDPublicKey)

}

type PushSubscriptionInput struct {

	Endpoint string `json:"endpoint"`
	Keys PushSubscriptionKeys `json:"keys"`

}

type PushSubscriptionKeys struct {

	P256dh string `json:"p256dh"`
	Auth string `json:"auth"`

}

func (s *PushService) UpsertSubscription(ctx context.Context, userID string, input PushSubscriptionInput) error {

	if !s.Configured() {

		return ErrPushNotConfigured

	}

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return err

	}

	endpoint := strings.TrimSpace(input.Endpoint)
	p256dh := strings.TrimSpace(input.Keys.P256dh)
	auth := strings.TrimSpace(input.Keys.Auth)

	if endpoint == "" || p256dh == "" || auth == "" {

		return ErrInvalidPushSubscription

	}

	now := time.Now()

	_, err = s.db.PushSubscriptions().UpdateOne(ctx, bson.M{

		"userId": oid,
		"endpoint": endpoint,

	}, bson.M{

		"$set": bson.M{

			"p256dh": p256dh,
			"auth": auth,
			"updatedAt": now,

		},
		"$setOnInsert": bson.M{

			"userId": oid,
			"endpoint": endpoint,

		},

	}, options.Update().SetUpsert(true))

	return err

}

func (s *PushService) DeleteSubscription(ctx context.Context, userID, endpoint string) error {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return err

	}

	endpoint = strings.TrimSpace(endpoint)

	if endpoint == "" {

		return ErrInvalidPushSubscription

	}

	_, err = s.db.PushSubscriptions().DeleteOne(ctx, bson.M{"userId": oid, "endpoint": endpoint})

	return err

}

func (s *PushService) DeleteEndpoint(ctx context.Context, endpoint string) {

	endpoint = strings.TrimSpace(endpoint)

	if endpoint == "" {

		return

	}

	_, _ = s.db.PushSubscriptions().DeleteMany(ctx, bson.M{"endpoint": endpoint})

}

func (s *PushService) ListForUser(ctx context.Context, userID primitive.ObjectID) ([]models.PushSubscription, error) {

	cur, err := s.db.PushSubscriptions().Find(ctx, bson.M{"userId": userID})

	if err != nil {

		return nil, err

	}

	defer cur.Close(ctx)

	var out []models.PushSubscription

	if err := cur.All(ctx, &out); err != nil {

		return nil, err

	}

	return out, nil

}

type SportsPushPayload struct {

	Title string `json:"title"`
	Body string `json:"body"`
	URL string `json:"url"`
	Tag string `json:"tag"`

}

func (s *PushService) SendToUser(ctx context.Context, userID primitive.ObjectID, payload SportsPushPayload) error {

	if !s.Configured() {

		return ErrPushNotConfigured

	}

	subs, err := s.ListForUser(ctx, userID)

	if err != nil {

		return err

	}

	if len(subs) == 0 {

		return nil

	}

	body, err := json.Marshal(payload)

	if err != nil {

		return err

	}

	subject := strings.TrimSpace(s.cfg.VAPIDSubject)

	if subject == "" {

		subject = "mailto:alerts@localhost"

	}

	for _, sub := range subs {

		select {

		case <-ctx.Done():

			return ctx.Err()

		default:

		}

		resp, err := webpush.SendNotificationWithContext(ctx, body, &webpush.Subscription{

			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{

				P256dh: sub.P256dh,
				Auth: sub.Auth,

			},

		}, &webpush.Options{

			Subscriber: subject,
			VAPIDPublicKey: s.cfg.VAPIDPublicKey,
			VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
			TTL: 120,
			Urgency: webpush.UrgencyHigh,

		})

		if err != nil {

			log.Printf("push: send failed endpoint=%s err=%v", truncateEndpoint(sub.Endpoint), err)
			continue

		}

		status := 0

		if resp != nil {

			status = resp.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
			_ = resp.Body.Close()

		}

		if status == http.StatusGone || status == http.StatusNotFound {

			s.DeleteEndpoint(ctx, sub.Endpoint)
			continue

		}

		if status >= 400 {

			log.Printf("push: send status=%d endpoint=%s", status, truncateEndpoint(sub.Endpoint))

		}

	}

	return nil

}

func truncateEndpoint(endpoint string) string {

	if len(endpoint) <= 64 {

		return endpoint

	}

	return endpoint[:64] + "…"

}
