package playback

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"streamly/internal/config"

	"golang.org/x/sync/singleflight"
)

const proxyTokenCacheMax = 4096

type ProxyEntry struct {

	Token string
	TargetURL string
	Referer string
	RequestHeaders map[string]string
	ExpiresAt time.Time

}

type proxyTokenCacheEntry struct {

	token string
	expiresAt time.Time

}

type ProxyService struct {

	ttl time.Duration
	client *http.Client

	tokenMu sync.Mutex
	tokenByKey map[string]proxyTokenCacheEntry
	entryByToken map[string]ProxyEntry
	tokenGroup singleflight.Group

}

func NewProxyService(cfg *config.Config) *ProxyService {

	checkRedirect := func(req *http.Request, via []*http.Request) error {

		if len(via) >= 5 {

			return errors.New("too many redirects")

		}

		return nil

	}

	transport := &http.Transport{

		Proxy: http.ProxyFromEnvironment,

		MaxIdleConns: 64,
		MaxIdleConnsPerHost: 16,

		IdleConnTimeout: 90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,

	}

	svc := &ProxyService{

		ttl: cfg.ProxyTokenTTL,

		tokenByKey: make(map[string]proxyTokenCacheEntry),
		entryByToken: make(map[string]ProxyEntry),

		client: &http.Client{

			Transport: transport,
			Timeout: 0,
			CheckRedirect: checkRedirect,

		},

	}

	return svc

}

type ProxySession struct {

	Token string `json:"token"`
	ProxyPath string `json:"proxyPath"`

	IsHLS bool `json:"isHls"`

}

func (s *ProxyService) CreateSessionWithHeaders(ctx context.Context, targetURL string, headers map[string]string, isHLS bool) (*ProxySession, error) {

	targetURL = strings.TrimSpace(targetURL)

	if targetURL == "" {

		return nil, errors.New("empty stream url")

	}

	referer := headers["Referer"]

	token, err := s.getOrCreateTokenWithHeaders(targetURL, referer, headers)

	if err != nil {

		return nil, err

	}

	return &ProxySession{

		Token: token,
		ProxyPath: "/api/proxy/" + token,
		IsHLS: isHLS,

	}, nil

}

func (s *ProxyService) ResolveToken(token string) (*ProxyEntry, error) {

	s.tokenMu.Lock()

	defer s.tokenMu.Unlock()

	entry, ok := s.entryByToken[token]

	if !ok || time.Now().After(entry.ExpiresAt) {

		if ok {

			delete(s.entryByToken, token)

		}

		return nil, errors.New("stream session expired or not found")

	}

	// Sliding expiry: an AirPlay receiver may pull the same playlist for hours,
	// and eviction of a token it is still using stops playback on the TV.
	entry.ExpiresAt = time.Now().Add(s.ttl)
	s.entryByToken[token] = entry

	return &entry, nil

}

func (s *ProxyService) Fetch(ctx context.Context, entry *ProxyEntry, method string, incoming http.Header) (*http.Response, error) {

	if method != http.MethodHead {

		method = http.MethodGet

	}

	req, err := http.NewRequestWithContext(ctx, method, entry.TargetURL, nil)

	if err != nil {

		return nil, err

	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	req.Header.Set("Accept", "*/*")

	if entry.Referer != "" {

		req.Header.Set("Referer", entry.Referer)

	}

	for key, value := range entry.RequestHeaders {

		if value == "" || strings.EqualFold(key, "Referer") {

			continue

		}

		req.Header.Set(key, value)

	}

	if rangeHeader := incoming.Get("Range"); rangeHeader != "" {

		req.Header.Set("Range", rangeHeader)

	}

	if ifRange := incoming.Get("If-Range"); ifRange != "" {

		req.Header.Set("If-Range", ifRange)

	}

	return s.client.Do(req)

}
