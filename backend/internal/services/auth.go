package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"streamly/internal/config"
	"streamly/internal/database"
	"streamly/internal/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidAccessCode  = errors.New("invalid or expired access code")
	ErrAccessCodeExhausted = errors.New("access code has reached its usage limit")
)

type Claims struct {
	UserID  string `json:"userId"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"isAdmin"`
	jwt.RegisteredClaims
}

type AuthService struct {
	db     *database.DB
	secret []byte
	expiry time.Duration
	secure bool
	domain string
	bootstrap string
}

func NewAuthService(db *database.DB, cfg *config.Config) *AuthService {
	return &AuthService{
		db:        db,
		secret:    []byte(cfg.JWTSecret),
		expiry:    cfg.JWTExpiry,
		secure:    cfg.CookieSecure,
		domain:    cfg.CookieDomain,
		bootstrap: cfg.BootstrapCode,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, accessCode string) (*models.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(password) < 8 {
		return nil, "", errors.New("email and password (min 8 chars) required")
	}

	var existing models.User
	err := s.db.Users().FindOne(ctx, bson.M{"email": email}).Decode(&existing)
	if err == nil {
		return nil, "", ErrEmailTaken
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, "", err
	}

	isBootstrap := s.bootstrap != "" && accessCode == s.bootstrap
	if !isBootstrap {
		if err := s.consumeAccessCode(ctx, accessCode); err != nil {
			return nil, "", err
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	user := models.User{
		Email:        email,
		PasswordHash: string(hash),
		IsAdmin:      isBootstrap,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	res, err := s.db.Users().InsertOne(ctx, user)
	if err != nil {
		return nil, "", err
	}
	user.ID = res.InsertedID.(primitive.ObjectID)

	settings := models.UserSettings{
		UserID:          user.ID,
		PreferredHeight: 1080,
		AutoPlayNext:    true,
		SkipIntro:       true,
		AmbienceEnabled: true,
		UpdatedAt:       now,
	}
	if _, err := s.db.Settings().InsertOne(ctx, settings); err != nil {
		return nil, "", err
	}

	token, err := s.issueToken(user)
	return &user, token, err
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	err := s.db.Users().FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.issueToken(user)
	return &user, token, err
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := s.db.Users().FindOne(ctx, bson.M{"_id": oid}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) issueToken(user models.User) (string, error) {
	claims := Claims{
		UserID:  user.ID.Hex(),
		Email:   user.Email,
		IsAdmin: user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (s *AuthService) CookieSettings() (secure bool, domain string) {
	return s.secure, s.domain
}

func (s *AuthService) consumeAccessCode(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return ErrInvalidAccessCode
	}

	var entry models.AccessCode
	err := s.db.AccessCodes().FindOne(ctx, bson.M{"code": code}).Decode(&entry)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrInvalidAccessCode
		}
		return err
	}

	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		return ErrInvalidAccessCode
	}
	if entry.MaxUses > 0 && entry.Uses >= entry.MaxUses {
		return ErrAccessCodeExhausted
	}

	_, err = s.db.AccessCodes().UpdateOne(ctx, bson.M{"_id": entry.ID}, bson.M{"$inc": bson.M{"uses": 1}})
	return err
}

func (s *AuthService) CreateAccessCode(ctx context.Context, creatorID string, maxUses int, expiresAt *time.Time) (*models.AccessCode, error) {
	oid, err := primitive.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, err
	}

	code, err := randomCode(16)
	if err != nil {
		return nil, err
	}

	entry := models.AccessCode{
		Code:      code,
		CreatedBy: oid,
		MaxUses:   maxUses,
		Uses:      0,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	res, err := s.db.AccessCodes().InsertOne(ctx, entry)
	if err != nil {
		return nil, err
	}
	entry.ID = res.InsertedID.(primitive.ObjectID)
	return &entry, nil
}

func (s *AuthService) ListAccessCodes(ctx context.Context) ([]models.AccessCode, error) {
	cur, err := s.db.AccessCodes().Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var codes []models.AccessCode
	if err := cur.All(ctx, &codes); err != nil {
		return nil, err
	}
	return codes, nil
}

func (s *AuthService) DeleteAccessCode(ctx context.Context, code string) error {
	res, err := s.db.AccessCodes().DeleteOne(ctx, bson.M{"code": code})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func randomCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}