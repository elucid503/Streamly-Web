package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimitConfig struct {

	Name string

	RequestsPerMinute int
	Burst int

	KeyFunc func(*gin.Context) string

}

type limiterEntry struct {

	limiter *rate.Limiter
	lastSeen time.Time

}

type rateLimiterStore struct {

	cfg RateLimitConfig

	mu sync.Mutex
	visitors map[string]*limiterEntry

}

func newRateLimiterStore(cfg RateLimitConfig) *rateLimiterStore {

	if cfg.RequestsPerMinute <= 0 {

		cfg.RequestsPerMinute = 60

	}

	if cfg.Burst <= 0 {

		cfg.Burst = cfg.RequestsPerMinute / 4

		if cfg.Burst < 1 {

			cfg.Burst = 1

		}

	}

	if cfg.KeyFunc == nil {

		cfg.KeyFunc = UserOrIPKey

	}

	store := &rateLimiterStore{

		cfg: cfg,
		visitors: make(map[string]*limiterEntry),

	}

	go store.cleanupLoop()

	return store

}

func (s *rateLimiterStore) allow(key string) bool {

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.visitors[key]

	if !ok {

		rps := rate.Limit(float64(s.cfg.RequestsPerMinute) / 60.0)

		entry = &limiterEntry{

			limiter: rate.NewLimiter(rps, s.cfg.Burst),
			lastSeen: time.Now(),

		}

		s.visitors[key] = entry

	} else {

		entry.lastSeen = time.Now()

	}

	return entry.limiter.Allow()

}

func (s *rateLimiterStore) cleanupLoop() {

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {

		s.mu.Lock()

		cutoff := time.Now().Add(-10 * time.Minute)

		for key, entry := range s.visitors {

			if entry.lastSeen.Before(cutoff) {

				delete(s.visitors, key)

			}

		}

		s.mu.Unlock()

	}

}

func UserOrIPKey(c *gin.Context) string {

	if userID, ok := c.Get(UserIDKey); ok {

		if id, ok := userID.(string); ok && id != "" {

			return "user:" + id

		}

	}

	return "ip:" + c.ClientIP()

}

func IPKey(c *gin.Context) string {

	return "ip:" + c.ClientIP()

}

func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {

	store := newRateLimiterStore(cfg)

	return func(c *gin.Context) {

		key := store.cfg.KeyFunc(c)

		if key == "" {

			key = "ip:" + c.ClientIP()

		}

		if store.cfg.Name != "" {

			key = store.cfg.Name + ":" + key

		}

		if !store.allow(key) {

			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})

			return

		}

		c.Next()

	}

}

var (

	AuthRateLimit = RateLimit(RateLimitConfig{

		Name: "auth",
		RequestsPerMinute: 20,
		Burst: 8,
		KeyFunc: IPKey,

	})

	SearchRateLimit = RateLimit(RateLimitConfig{

		Name: "search",
		RequestsPerMinute: 60,
		Burst: 20,

	})

	CatalogRateLimit = RateLimit(RateLimitConfig{

		Name: "catalog",
		RequestsPerMinute: 180,
		Burst: 45,

	})

	StreamRateLimit = RateLimit(RateLimitConfig{

		Name: "stream",
		RequestsPerMinute: 90,
		Burst: 25,

	})

)