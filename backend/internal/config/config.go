package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {

	Port string

	MongoURI  string

	JWTSecret string
	JWTExpiry time.Duration

	CookieSecure bool
	CookieDomain string
	FrontendOrigin string

	FebboxCookie string
	IntroDBKey string
	TMDBAPIKey string
	SubDLAPIKey  string
	TVBaseURL string

	ChildMode string

	BootstrapCode  string
	DefaultQuality int

	ProxyTokenTTL time.Duration
	CatalogCacheTTL time.Duration
	CatalogCacheFile string
	SubtitleCacheTTL time.Duration

	StaticDir string

}

func Load() (*Config, error) {

	_ = godotenv.Load()

	cfg := &Config{

		Port: envOr("PORT", "8080"),
		MongoURI: os.Getenv("MONGO_URI"),
		JWTSecret: envOr("JWT_SECRET", "change-me-in-production"),
		JWTExpiry: durationOr("JWT_EXPIRY", 7*24*time.Hour),

		CookieSecure: boolOr("COOKIE_SECURE", false),
		CookieDomain: os.Getenv("COOKIE_DOMAIN"),
		FrontendOrigin: envOr("FRONTEND_ORIGIN", "http://localhost:5173"),

		FebboxCookie: os.Getenv("FEBBOX_UI_COOKIE"),
		IntroDBKey: os.Getenv("INTRODB_API_KEY"),
		TMDBAPIKey: os.Getenv("TMDB_API_KEY"),
		SubDLAPIKey: os.Getenv("SUBDL_API_KEY"),
		TVBaseURL: os.Getenv("TV_BASE_URL"),

		ChildMode: envOr("CHILD_MODE", "0"),

		BootstrapCode: os.Getenv("BOOTSTRAP_ACCESS_CODE"),
		DefaultQuality: intOr("DEFAULT_QUALITY", 1080),

		ProxyTokenTTL: durationOr("PROXY_TOKEN_TTL", 4*time.Hour),
		CatalogCacheTTL: durationOr("CATALOG_CACHE_TTL", time.Hour),
		CatalogCacheFile: envOr("CATALOG_CACHE_FILE", "data/catalog.cache.json"),
		SubtitleCacheTTL: durationOr("SUBTITLE_CACHE_TTL", 15*time.Minute),

		StaticDir: envOr("STATIC_DIR", "../frontend/dist"),

	}

	return cfg, nil

}

func envOr(key, fallback string) string {

	if v := strings.TrimSpace(os.Getenv(key)); v != "" {

		return v

	}

	return fallback

}

func intOr(key string, fallback int) int {

	if v := strings.TrimSpace(os.Getenv(key)); v != "" {

		if n, err := strconv.Atoi(v); err == nil {

			return n

		}

	}

	return fallback

}

func boolOr(key string, fallback bool) bool {

	if v := strings.TrimSpace(os.Getenv(key)); v != "" {

		parsed, err := strconv.ParseBool(v)

		if err == nil {

			return parsed

		}

	}

	return fallback

}

func durationOr(key string, fallback time.Duration) time.Duration {

	if v := strings.TrimSpace(os.Getenv(key)); v != "" {

		if d, err := time.ParseDuration(v); err == nil {

			return d

		}

	}

	return fallback

}
