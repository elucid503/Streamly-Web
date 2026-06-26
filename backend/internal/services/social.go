package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"streamly/internal/database"
	"streamly/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (

	ErrSocialSelf = errors.New("cannot perform this action on yourself")
	ErrSocialDuplicate = errors.New("request already exists")
	ErrSocialForbidden = errors.New("forbidden")
	ErrSocialBadInput = errors.New("invalid input")

)

// SSE hub

type SSEEvent struct {

	Type string `json:"type"`

}

type SocialHub struct {

	mu sync.RWMutex
	clients map[string]map[chan SSEEvent]struct{}

}

func NewSocialHub() *SocialHub {

	return &SocialHub{clients: make(map[string]map[chan SSEEvent]struct{})}

}

func (h *SocialHub) Subscribe(userID string) chan SSEEvent {

	ch := make(chan SSEEvent, 4)

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[userID] == nil {

		h.clients[userID] = make(map[chan SSEEvent]struct{})

	}

	h.clients[userID][ch] = struct{}{}

	return ch

}

func (h *SocialHub) Unsubscribe(userID string, ch chan SSEEvent) {

	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients[userID], ch)

	if len(h.clients[userID]) == 0 {

		delete(h.clients, userID)

	}

}

func (h *SocialHub) Notify(userID string, event SSEEvent) {

	h.mu.RLock()

	chans := make([]chan SSEEvent, 0, len(h.clients[userID]))

	for ch := range h.clients[userID] {

		chans = append(chans, ch)

	}

	h.mu.RUnlock()

	for _, ch := range chans {

		select {

		case ch <- event:

		default:

		}

	}

}

const (

	FriendStatusNone = "none"
	FriendStatusPendingSent = "pending_sent"
	FriendStatusPendingReceived = "pending_received"
	FriendStatusFriends = "friends"

	FriendRequestPending = "pending"
	FriendRequestAccepted = "accepted"

)

var validBanners = map[string]bool{

	"aurora":   true,
	"sunset":   true,
	"ocean":    true,
	"forest":   true,
	"midnight": true,
	"rose":     true,
	"ember":    true,
	"slate":    true,
	"nebula":   true,
	"cosmos":   true,

}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func boolPtr(b bool) *bool { return &b }

type ProfileUpdate struct {

	DisplayName    string               `json:"displayName"`
	Bio            string               `json:"bio"`
	AccentColor    string               `json:"accentColor"`
	Banner         string               `json:"banner"`
	FavoriteMovies []models.ProfileMedia `json:"favoriteMovies"`
	FavoriteShows  []models.ProfileMedia `json:"favoriteShows"`
	HistoryVisible  *bool `json:"historyVisible"`
	DiscoverVisible *bool `json:"discoverVisible"`
}

type UserSummary struct {

	UserID       string `json:"userId"`
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
	AccentColor  string `json:"accentColor"`
	Banner       string `json:"banner"`
	FriendStatus string `json:"friendStatus"`

}

type PublicProfileResponse struct {

	UserID         string                    `json:"userId"`
	Email          string                    `json:"email"`
	DisplayName    string                    `json:"displayName"`
	Bio            string                    `json:"bio"`
	AccentColor    string                    `json:"accentColor"`
	Banner         string                    `json:"banner"`
	FavoriteMovies []models.ProfileMedia     `json:"favoriteMovies"`
	FavoriteShows  []models.ProfileMedia     `json:"favoriteShows"`
	RecentHistory  []models.WatchHistoryItem `json:"recentHistory"`
	FriendStatus   string                    `json:"friendStatus"`

}

type FriendRequestResponse struct {

	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	AccentColor string    `json:"accentColor"`
	Banner      string    `json:"banner"`
	CreatedAt   time.Time `json:"createdAt"`
	Direction   string    `json:"direction"`

}

type SocialService struct {

	db  *database.DB
	hub *SocialHub

}

func NewSocialService(db *database.DB, hub *SocialHub) *SocialService {

	return &SocialService{db: db, hub: hub}

}

func (s *SocialService) Hub() *SocialHub {

	return s.hub

}

func defaultDisplayName(email string) string {

	if idx := strings.Index(email, "@"); idx > 0 {

		return email[:idx]

	}

	return email

}

func (s *SocialService) getOrCreateProfile(ctx context.Context, userOID primitive.ObjectID, email string) (*models.UserProfile, error) {

	var profile models.UserProfile

	err := s.db.Profiles().FindOne(ctx, bson.M{"userId": userOID}).Decode(&profile)

	if err == nil {

		return &profile, nil

	}

	if !errors.Is(err, mongo.ErrNoDocuments) {

		return nil, err

	}

	now := time.Now()

	profile = models.UserProfile{

		UserID:         userOID,

		DisplayName:    defaultDisplayName(email),
		Bio:            "",

		AccentColor:    "#6366f1",
		Banner:         "aurora",

		FavoriteMovies:  []models.ProfileMedia{},
		FavoriteShows:   []models.ProfileMedia{},

		HistoryVisible:  true,
		DiscoverVisible: boolPtr(true),

		UpdatedAt:       now,

	}

	result, err := s.db.Profiles().InsertOne(ctx, profile)

	if err != nil {

		// Race condition: another request inserted first

		if mongo.IsDuplicateKeyError(err) {

			err2 := s.db.Profiles().FindOne(ctx, bson.M{"userId": userOID}).Decode(&profile)

			return &profile, err2

		}

		return nil, err

	}

	profile.ID = result.InsertedID.(primitive.ObjectID)

	return &profile, nil

}

func (s *SocialService) friendStatus(ctx context.Context, viewerOID, targetOID primitive.ObjectID) (string, *models.FriendRequest, error) {

	filter := bson.M{

		"$or": bson.A{

			bson.M{"fromId": viewerOID, "toId": targetOID},
			bson.M{"fromId": targetOID, "toId": viewerOID},

		},

	}

	var req models.FriendRequest

	err := s.db.FriendRequests().FindOne(ctx, filter).Decode(&req)

	if errors.Is(err, mongo.ErrNoDocuments) {

		return FriendStatusNone, nil, nil

	}

	if err != nil {

		return "", nil, err

	}

	if req.Status == FriendRequestAccepted {

		return FriendStatusFriends, &req, nil

	}

	if req.FromID == viewerOID {

		return FriendStatusPendingSent, &req, nil

	}

	return FriendStatusPendingReceived, &req, nil

}

func (s *SocialService) GetMyProfile(ctx context.Context, userID string) (*models.UserProfile, error) {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	var user models.User

	err = s.db.Users().FindOne(ctx, bson.M{"_id": oid}).Decode(&user)

	if err != nil {

		return nil, err

	}

	return s.getOrCreateProfile(ctx, oid, user.Email)

}

func (s *SocialService) UpdateProfile(ctx context.Context, userID string, input ProfileUpdate) (*models.UserProfile, error) {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	var user models.User

	err = s.db.Users().FindOne(ctx, bson.M{"_id": oid}).Decode(&user)

	if err != nil {

		return nil, err

	}

	if _, err := s.getOrCreateProfile(ctx, oid, user.Email); err != nil {

		return nil, err

	}

	set := bson.M{

		"updatedAt": time.Now(),
	}

	displayName := strings.TrimSpace(input.DisplayName)

	if displayName != "" {

		if len(displayName) > 32 {

			return nil, ErrSocialBadInput

		}

		set["displayName"] = displayName

	}

	bio := strings.TrimSpace(input.Bio)

	if len(bio) > 160 {

		return nil, ErrSocialBadInput

	}

	set["bio"] = bio

	if input.AccentColor != "" {

		if !hexColorRe.MatchString(input.AccentColor) {

			return nil, ErrSocialBadInput

		}

		set["accentColor"] = input.AccentColor

	}

	if input.Banner != "" {

		if !validBanners[input.Banner] {

			return nil, ErrSocialBadInput

		}

		set["banner"] = input.Banner

	}

	if input.HistoryVisible != nil {

		set["historyVisible"] = *input.HistoryVisible

	}

	if input.DiscoverVisible != nil {

		set["discoverVisible"] = *input.DiscoverVisible

	}

	movies := input.FavoriteMovies

	if movies == nil {

		movies = []models.ProfileMedia{}

	}

	if len(movies) > 3 {

		movies = movies[:3]

	}

	set["favoriteMovies"] = movies

	shows := input.FavoriteShows

	if shows == nil {

		shows = []models.ProfileMedia{}

	}

	if len(shows) > 3 {

		shows = shows[:3]

	}

	set["favoriteShows"] = shows

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updated models.UserProfile

	err = s.db.Profiles().FindOneAndUpdate(ctx, bson.M{"userId": oid}, bson.M{"$set": set}, opts).Decode(&updated)

	if err != nil {

		return nil, err

	}

	return &updated, nil

}

func (s *SocialService) GetPublicProfile(ctx context.Context, viewerID, targetID string) (*PublicProfileResponse, error) {

	viewerOID, err := primitive.ObjectIDFromHex(viewerID)

	if err != nil {

		return nil, err

	}

	targetOID, err := primitive.ObjectIDFromHex(targetID)

	if err != nil {

		return nil, err

	}

	var targetUser models.User

	err = s.db.Users().FindOne(ctx, bson.M{"_id": targetOID}).Decode(&targetUser)

	if err != nil {

		return nil, err

	}

	profile, err := s.getOrCreateProfile(ctx, targetOID, targetUser.Email)

	if err != nil {

		return nil, err

	}

	status, _, err := s.friendStatus(ctx, viewerOID, targetOID)

	if err != nil {

		return nil, err

	}

	resp := &PublicProfileResponse{

		UserID:         targetUser.ID.Hex(),

		Email:          targetUser.Email,
		DisplayName:    profile.DisplayName,
		Bio:            profile.Bio,

		AccentColor:    profile.AccentColor,
		Banner:         profile.Banner,

		FavoriteMovies: profile.FavoriteMovies,
		FavoriteShows:  profile.FavoriteShows,

		RecentHistory:  []models.WatchHistoryItem{},

		FriendStatus:   status,

	}

	if profile.FavoriteMovies == nil {

		resp.FavoriteMovies = []models.ProfileMedia{}

	}

	if profile.FavoriteShows == nil {

		resp.FavoriteShows = []models.ProfileMedia{}

	}

	if profile.HistoryVisible && status == FriendStatusFriends {

		cur, err := s.db.History().Find(ctx, bson.M{"userId": targetOID},
			options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}).SetLimit(10))

		if err == nil {

			defer cur.Close(ctx)

			var items []models.WatchHistoryItem

			if err := cur.All(ctx, &items); err == nil {

				resp.RecentHistory = items

			}

		}

	}

	return resp, nil

}

func (s *SocialService) SearchUsers(ctx context.Context, viewerID, query string) ([]UserSummary, error) {

	viewerOID, err := primitive.ObjectIDFromHex(viewerID)

	if err != nil {

		return nil, err

	}

	query = strings.TrimSpace(query)

	var userFilter bson.M

	if query == "" {

		userFilter = bson.M{"_id": bson.M{"$ne": viewerOID}}

	} else {

		userFilter = bson.M{

			"_id":   bson.M{"$ne": viewerOID},
			"email": bson.M{"$regex": regexp.QuoteMeta(query), "$options": "i"},
		}

	}

	cur, err := s.db.Users().Find(ctx, userFilter, options.Find().SetLimit(30).SetSort(bson.D{{Key: "createdAt", Value: -1}}))

	if err != nil {

		return nil, err

	}

	defer cur.Close(ctx)

	var users []models.User

	if err := cur.All(ctx, &users); err != nil {

		return nil, err

	}

	if len(users) == 0 {

		return []UserSummary{}, nil

	}

	userIDs := make([]primitive.ObjectID, len(users))

	for i, u := range users {

		userIDs[i] = u.ID

	}

	profileCur, err := s.db.Profiles().Find(ctx, bson.M{"userId": bson.M{"$in": userIDs}})

	if err != nil {

		return nil, err

	}

	defer profileCur.Close(ctx)

	var profiles []models.UserProfile

	_ = profileCur.All(ctx, &profiles)

	profileMap := make(map[primitive.ObjectID]*models.UserProfile, len(profiles))

	for i := range profiles {

		profileMap[profiles[i].UserID] = &profiles[i]

	}

	// Load all requests involving viewer in one query

	reqCur, err := s.db.FriendRequests().Find(ctx, bson.M{

		"$or": bson.A{

			bson.M{"fromId": viewerOID},
			bson.M{"toId": viewerOID},
		},
	})

	if err != nil {

		return nil, err

	}

	defer reqCur.Close(ctx)

	var allReqs []models.FriendRequest

	_ = reqCur.All(ctx, &allReqs)

	statusFor := func(targetOID primitive.ObjectID) string {

		for _, r := range allReqs {

			if (r.FromID == viewerOID && r.ToID == targetOID) || (r.FromID == targetOID && r.ToID == viewerOID) {

				if r.Status == FriendRequestAccepted {

					return FriendStatusFriends

				}

				if r.FromID == viewerOID {

					return FriendStatusPendingSent

				}

				return FriendStatusPendingReceived

			}

		}

		return FriendStatusNone

	}

	summaries := make([]UserSummary, 0, len(users))

	for _, u := range users {

		p := profileMap[u.ID]

		// Skip users who have explicitly opted out of discovery
		if p != nil && p.DiscoverVisible != nil && !*p.DiscoverVisible {

			continue

		}

		displayName := defaultDisplayName(u.Email)
		accentColor := "#6366f1"
		banner := "aurora"

		if p != nil {

			displayName = p.DisplayName
			accentColor = p.AccentColor
			banner = p.Banner

		}

		summaries = append(summaries, UserSummary{

			UserID:       u.ID.Hex(),

			Email:        u.Email,
			DisplayName:  displayName,

			AccentColor:  accentColor,
			Banner:       banner,

			FriendStatus: statusFor(u.ID),

		})

	}

	return summaries, nil

}

func (s *SocialService) ListFriends(ctx context.Context, userID string) ([]UserSummary, error) {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	cur, err := s.db.FriendRequests().Find(ctx, bson.M{

		"status": FriendRequestAccepted,
		"$or": bson.A{

			bson.M{"fromId": oid},
			bson.M{"toId": oid},
		},

	})

	if err != nil {

		return nil, err

	}

	defer cur.Close(ctx)

	var reqs []models.FriendRequest

	if err := cur.All(ctx, &reqs); err != nil {

		return nil, err

	}

	if len(reqs) == 0 {

		return []UserSummary{}, nil

	}

	friendIDs := make([]primitive.ObjectID, 0, len(reqs))

	for _, r := range reqs {

		if r.FromID == oid {

			friendIDs = append(friendIDs, r.ToID)

		} else {

			friendIDs = append(friendIDs, r.FromID)

		}

	}

	userCur, err := s.db.Users().Find(ctx, bson.M{"_id": bson.M{"$in": friendIDs}})

	if err != nil {

		return nil, err

	}

	defer userCur.Close(ctx)

	var users []models.User

	if err := userCur.All(ctx, &users); err != nil {

		return nil, err

	}

	profileCur, err := s.db.Profiles().Find(ctx, bson.M{"userId": bson.M{"$in": friendIDs}})

	if err != nil {

		return nil, err

	}

	defer profileCur.Close(ctx)

	var profiles []models.UserProfile

	_ = profileCur.All(ctx, &profiles)

	profileMap := make(map[primitive.ObjectID]*models.UserProfile)

	for i := range profiles {

		profileMap[profiles[i].UserID] = &profiles[i]

	}

	summaries := make([]UserSummary, 0, len(users))

	for _, u := range users {

		p := profileMap[u.ID]

		displayName := defaultDisplayName(u.Email)
		accentColor := "#6366f1"
		banner := "aurora"

		if p != nil {

			displayName = p.DisplayName
			accentColor = p.AccentColor
			banner = p.Banner

		}

		summaries = append(summaries, UserSummary{

			UserID:       u.ID.Hex(),

			Email:        u.Email,
			DisplayName:  displayName,

			AccentColor:  accentColor,
			Banner:       banner,

			FriendStatus: FriendStatusFriends,

		})

	}

	return summaries, nil

}

func (s *SocialService) ListRequests(ctx context.Context, userID string) ([]FriendRequestResponse, error) {

	oid, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return nil, err

	}

	cur, err := s.db.FriendRequests().Find(ctx, bson.M{

		"status": FriendRequestPending,
		"$or": bson.A{

			bson.M{"fromId": oid},
			bson.M{"toId": oid},
		},

	}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))

	if err != nil {

		return nil, err

	}

	defer cur.Close(ctx)

	var reqs []models.FriendRequest

	if err := cur.All(ctx, &reqs); err != nil {

		return nil, err

	}

	if len(reqs) == 0 {

		return []FriendRequestResponse{}, nil

	}

	otherIDs := make([]primitive.ObjectID, 0, len(reqs))

	for _, r := range reqs {

		if r.FromID == oid {

			otherIDs = append(otherIDs, r.ToID)

		} else {

			otherIDs = append(otherIDs, r.FromID)

		}

	}

	userCur, err := s.db.Users().Find(ctx, bson.M{"_id": bson.M{"$in": otherIDs}})

	if err != nil {

		return nil, err

	}

	defer userCur.Close(ctx)

	var users []models.User

	if err := userCur.All(ctx, &users); err != nil {

		return nil, err

	}

	userMap := make(map[primitive.ObjectID]models.User)

	for _, u := range users {

		userMap[u.ID] = u

	}

	profileCur, err := s.db.Profiles().Find(ctx, bson.M{"userId": bson.M{"$in": otherIDs}})

	if err != nil {

		return nil, err

	}

	defer profileCur.Close(ctx)

	var profiles []models.UserProfile

	_ = profileCur.All(ctx, &profiles)

	profileMap := make(map[primitive.ObjectID]*models.UserProfile)

	for i := range profiles {

		profileMap[profiles[i].UserID] = &profiles[i]

	}

	responses := make([]FriendRequestResponse, 0, len(reqs))

	for _, r := range reqs {

		otherOID := r.ToID

		direction := "outgoing"

		if r.ToID == oid {

			otherOID = r.FromID
			direction = "incoming"

		}

		u, ok := userMap[otherOID]

		if !ok {

			continue

		}

		p := profileMap[otherOID]

		displayName := defaultDisplayName(u.Email)
		accentColor := "#6366f1"
		banner := "aurora"

		if p != nil {

			displayName = p.DisplayName
			accentColor = p.AccentColor
			banner = p.Banner

		}

		responses = append(responses, FriendRequestResponse{

			ID:          r.ID.Hex(),
			UserID:      u.ID.Hex(),
			Email:       u.Email,
			DisplayName: displayName,
			AccentColor: accentColor,
			Banner:      banner,
			CreatedAt:   r.CreatedAt,
			Direction:   direction,

		})

	}

	return responses, nil

}

func (s *SocialService) SendRequest(ctx context.Context, fromID, toID string) error {

	fromOID, err := primitive.ObjectIDFromHex(fromID)

	if err != nil {

		return err

	}

	toOID, err := primitive.ObjectIDFromHex(toID)

	if err != nil {

		return err

	}

	if fromOID == toOID {

		return ErrSocialSelf

	}

	// Verify target exists

	count, err := s.db.Users().CountDocuments(ctx, bson.M{"_id": toOID})

	if err != nil {

		return err

	}

	if count == 0 {

		return mongo.ErrNoDocuments

	}

	// Check no existing request

	existing, err := s.db.FriendRequests().CountDocuments(ctx, bson.M{

		"$or": bson.A{

			bson.M{"fromId": fromOID, "toId": toOID},
			bson.M{"fromId": toOID, "toId": fromOID},

		},

	})

	if err != nil {

		return err

	}

	if existing > 0 {

		return ErrSocialDuplicate

	}

	now := time.Now()

	_, err = s.db.FriendRequests().InsertOne(ctx, models.FriendRequest{

		FromID:    fromOID,
		ToID:      toOID,

		Status:    FriendRequestPending,

		CreatedAt: now,
		UpdatedAt: now,

	})

	if mongo.IsDuplicateKeyError(err) {

		return ErrSocialDuplicate

	}

	if err == nil {

		s.hub.Notify(toID, SSEEvent{Type: "friend_request"})

	}

	return err

}

func (s *SocialService) AcceptRequest(ctx context.Context, requestID, userID string) error {

	reqOID, err := primitive.ObjectIDFromHex(requestID)

	if err != nil {

		return err

	}

	userOID, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return err

	}

	var req models.FriendRequest

	err = s.db.FriendRequests().FindOneAndUpdate(ctx, bson.M{

		"_id":    reqOID,
		"toId":   userOID,
		"status": FriendRequestPending,

	}, bson.M{

		"$set": bson.M{

			"status":    FriendRequestAccepted,
			"updatedAt": time.Now(),
		},

	}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&req)

	if err != nil {

		if errors.Is(err, mongo.ErrNoDocuments) {

			return ErrSocialForbidden

		}

		return err

	}

	s.hub.Notify(req.FromID.Hex(), SSEEvent{Type: "request_accepted"})

	return nil

}

func (s *SocialService) DeleteRequest(ctx context.Context, requestID, userID string) error {

	reqOID, err := primitive.ObjectIDFromHex(requestID)

	if err != nil {

		return err

	}

	userOID, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return err

	}

	result, err := s.db.FriendRequests().DeleteOne(ctx, bson.M{

		"_id":    reqOID,
		"status": FriendRequestPending,

		"$or": bson.A{

			bson.M{"fromId": userOID},
			bson.M{"toId": userOID},

		},

	})

	if err != nil {

		return err

	}

	if result.DeletedCount == 0 {

		return ErrSocialForbidden

	}

	return nil

}

func (s *SocialService) RemoveFriend(ctx context.Context, userID, friendID string) error {

	userOID, err := primitive.ObjectIDFromHex(userID)

	if err != nil {

		return err

	}

	friendOID, err := primitive.ObjectIDFromHex(friendID)

	if err != nil {

		return err

	}

	_, err = s.db.FriendRequests().DeleteOne(ctx, bson.M{

		"status": FriendRequestAccepted,
		"$or": bson.A{

			bson.M{"fromId": userOID, "toId": friendOID},
			bson.M{"fromId": friendOID, "toId": userOID},
		},

	})

	return err

}
