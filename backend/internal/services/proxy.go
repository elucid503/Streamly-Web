package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"streamly/internal/config"
	"streamly/internal/database"
	"streamly/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/singleflight"
)

var hlsURIAttr = regexp.MustCompile(`URI="([^"]+)"`)

const proxyTokenCacheMax = 4096

type ProxyService struct {

	db *database.DB
	ttl time.Duration
	client *http.Client

	tokenMu sync.Mutex
	tokenByKey map[string]proxyTokenCacheEntry
	entryByToken map[string]models.ProxyToken
	tokenGroup singleflight.Group

}

type proxyTokenCacheEntry struct {

	token string
	expiresAt time.Time

}

func NewProxyService(db *database.DB, cfg *config.Config) *ProxyService {

	transport := &http.Transport{

		Proxy: http.ProxyFromEnvironment,

		MaxIdleConns: 64,
		MaxIdleConnsPerHost: 16,

		IdleConnTimeout: 90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,

	}

	return &ProxyService{

		db: db,
		ttl: cfg.ProxyTokenTTL,

		tokenByKey: make(map[string]proxyTokenCacheEntry),
		entryByToken: make(map[string]models.ProxyToken),

		client: &http.Client{

			Transport: transport,
			Timeout: 0,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {

				if len(via) >= 5 {

					return errors.New("too many redirects")

				}

				return nil

			},

		},

	}

}

type ProxySession struct {

	Token string `json:"token"`
	ProxyPath string `json:"proxyPath"`

	IsHLS bool `json:"isHls"`

}

func (s *ProxyService) CreateInlineSession(ctx context.Context, content []byte, contentType string) (*ProxySession, error) {

	if len(content) == 0 {

		return nil, errors.New("empty subtitle content")

	}

	token, err := randomToken(24)

	if err != nil {

		return nil, err

	}

	if strings.TrimSpace(contentType) == "" {

		contentType = "text/plain; charset=utf-8"

	}

	entry := models.ProxyToken{

		Token: token,
		InlineContent: content,
		InlineContentType: contentType,
		ExpiresAt: time.Now().Add(s.ttl),

	}

	if _, err := s.db.ProxyTokens().InsertOne(ctx, entry); err != nil {

		return nil, err

	}

	return &ProxySession{

		Token: token,
		ProxyPath: "/api/proxy/" + token,

	}, nil

}

func (s *ProxyService) CreateSession(ctx context.Context, targetURL, referer string, isHLS bool) (*ProxySession, error) {

	targetURL = strings.TrimSpace(targetURL)

	if targetURL == "" {

		return nil, errors.New("empty stream url")

	}

	token, err := s.getOrCreateURLToken(ctx, targetURL, referer)

	if err != nil {

		return nil, err

	}

	return &ProxySession{

		Token: token,
		ProxyPath: "/api/proxy/" + token,
		IsHLS: isHLS,

	}, nil

}

func (s *ProxyService) ResolveToken(ctx context.Context, token string) (*models.ProxyToken, error) {

	if entry, ok := s.cachedEntry(token); ok {

		return entry, nil

	}

	var entry models.ProxyToken

	err := s.db.ProxyTokens().FindOne(ctx, bson.M{"token": token, "expiresAt": bson.M{"$gt": time.Now()}}).Decode(&entry)

	if err != nil {

		return nil, err

	}

	s.cacheEntry(entry)

	return &entry, nil

}

func (s *ProxyService) Fetch(ctx context.Context, entry *models.ProxyToken, incoming http.Header) (*http.Response, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.TargetURL, nil)

	if err != nil {

		return nil, err

	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	req.Header.Set("Accept", "*/*")

	if entry.Referer != "" {

		req.Header.Set("Referer", entry.Referer)

	}

	if rangeHeader := incoming.Get("Range"); rangeHeader != "" {

		req.Header.Set("Range", rangeHeader)

	}

	if ifRange := incoming.Get("If-Range"); ifRange != "" {

		req.Header.Set("If-Range", ifRange)

	}

	return s.client.Do(req)

}

func (s *ProxyService) proxyMediaURL(ctx context.Context, base *url.URL, referer, baseProxyURL, raw string) (string, error) {

	trimmed := strings.TrimSpace(raw)

	if trimmed == "" || strings.HasPrefix(trimmed, "data:") {

		return trimmed, nil

	}

	resolved := resolveRelativeURL(base, trimmed)

	token, err := s.getOrCreateURLToken(ctx, resolved, referer)

	if err != nil {

		return "", err

	}

	return baseProxyURL + "/api/proxy/" + token, nil

}

func (s *ProxyService) RewritePlaylist(ctx context.Context, body []byte, entry *models.ProxyToken, baseProxyURL string) []byte {

	text := strings.ReplaceAll(string(body), "\r\n", "\n")

	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")

	base, _ := url.Parse(entry.TargetURL)

	out := make([]string, 0, len(lines))

	for _, line := range lines {

		trimmed := strings.TrimSpace(line)

		if trimmed == "" {

			out = append(out, "")

			continue

		}

		if strings.HasPrefix(trimmed, "#") {

			rewritten := hlsURIAttr.ReplaceAllStringFunc(line, func(match string) string {

				parts := hlsURIAttr.FindStringSubmatch(match)

				if len(parts) < 2 {

					return match

				}

				proxyURL, err := s.proxyMediaURL(ctx, base, entry.Referer, baseProxyURL, parts[1])

				if err != nil {

					return match

				}

				return `URI="` + proxyURL + `"`

			})

			out = append(out, rewritten)

			continue

		}

		proxyURL, err := s.proxyMediaURL(ctx, base, entry.Referer, baseProxyURL, trimmed)

		if err != nil {

			out = append(out, line)

			continue

		}

		out = append(out, proxyURL)

	}

	return []byte(strings.Join(out, "\n"))

}

func (s *ProxyService) getOrCreateURLToken(ctx context.Context, targetURL, referer string) (string, error) {

	key := proxyTokenKey(targetURL, referer)
	now := time.Now()

	s.tokenMu.Lock()

	if cached, ok := s.tokenByKey[key]; ok && now.Before(cached.expiresAt) {

		token := cached.token

		s.tokenMu.Unlock()

		return token, nil

	}

	s.tokenMu.Unlock()

	result, err, _ := s.tokenGroup.Do(key, func() (any, error) {

		s.tokenMu.Lock()

		if cached, ok := s.tokenByKey[key]; ok && time.Now().Before(cached.expiresAt) {

			token := cached.token

			s.tokenMu.Unlock()

			return token, nil

		}

		s.tokenMu.Unlock()

		token, err := randomToken(24)

		if err != nil {

			return "", err

		}

		entry := models.ProxyToken{

			Token: token,
			TargetURL: targetURL,
			Referer: referer,
			ExpiresAt: time.Now().Add(s.ttl),

		}

		if _, err := s.db.ProxyTokens().InsertOne(ctx, entry); err != nil {

			return "", err

		}

		s.tokenMu.Lock()

		s.pruneTokenCacheLocked(now)

		s.tokenByKey[key] = proxyTokenCacheEntry{

			token: token,
			expiresAt: entry.ExpiresAt,

		}

		s.entryByToken[token] = entry

		s.tokenMu.Unlock()

		return token, nil

	})

	if err != nil {

		return "", err

	}

	token := result.(string)

	return token, nil

}

func (s *ProxyService) cachedEntry(token string) (*models.ProxyToken, bool) {

	now := time.Now()

	s.tokenMu.Lock()

	defer s.tokenMu.Unlock()

	entry, ok := s.entryByToken[token]

	if !ok || now.After(entry.ExpiresAt) {

		if ok {

			delete(s.entryByToken, token)

		}

		return nil, false

	}

	return &entry, true

}

func (s *ProxyService) cacheEntry(entry models.ProxyToken) {

	s.tokenMu.Lock()

	defer s.tokenMu.Unlock()

	s.pruneTokenCacheLocked(time.Now())

	s.entryByToken[entry.Token] = entry

	if entry.TargetURL != "" {

		s.tokenByKey[proxyTokenKey(entry.TargetURL, entry.Referer)] = proxyTokenCacheEntry{

			token: entry.Token,
			expiresAt: entry.ExpiresAt,

		}

	}

}

func (s *ProxyService) pruneTokenCacheLocked(now time.Time) {

	if len(s.entryByToken) < proxyTokenCacheMax && len(s.tokenByKey) < proxyTokenCacheMax {

		return

	}

	for token, entry := range s.entryByToken {

		if now.After(entry.ExpiresAt) || len(s.entryByToken) > proxyTokenCacheMax {

			delete(s.entryByToken, token)

		}

	}

	for key, entry := range s.tokenByKey {

		if now.After(entry.expiresAt) || len(s.tokenByKey) > proxyTokenCacheMax {

			delete(s.tokenByKey, key)

		}

	}

}

func proxyTokenKey(targetURL, referer string) string {

	sum := sha256.Sum256([]byte(targetURL + "\x00" + referer))

	return hex.EncodeToString(sum[:])

}

func (s *ProxyService) AttachProxyURLs(ctx context.Context, stream *StreamDTO, referer, baseProxyURL string, force bool) error {

	if stream == nil || stream.URL == "" {

		return errors.New("empty stream")

	}

	if !force && !shouldProxyStream(stream.URL, stream.IsHLS) {

		return nil

	}

	session, err := s.CreateSession(ctx, stream.URL, referer, stream.IsHLS)

	if err != nil {

		return err

	}

	stream.ProxyURL = baseProxyURL + session.ProxyPath

	for i := range stream.Qualities {

		quality := &stream.Qualities[i]

		if quality.URL == stream.URL {

			quality.ProxyURL = stream.ProxyURL

		}

	}

	return nil

}

func shouldProxyStream(rawURL string, isHLS bool) bool {

	if isHLS || IsPlaylist("", rawURL) {

		return true

	}

	lower := strings.ToLower(strings.Split(rawURL, "?")[0])

	return strings.HasSuffix(lower, ".mkv") ||
		strings.HasSuffix(lower, ".avi") ||
		strings.HasSuffix(lower, ".wmv") ||
		strings.HasSuffix(lower, ".flv")

}

func (s *ProxyService) StreamQualities(qualities []QualityDTO, bestURL, referer string, isHLS bool, baseProxyURL string) (*StreamDTO, error) {

	if !shouldProxyStream(bestURL, isHLS) {

		return &StreamDTO{

			Qualities: qualities,
			URL: bestURL,
			IsHLS: isHLS,

		}, nil

	}

	session, err := s.CreateSession(context.Background(), bestURL, referer, isHLS)

	if err != nil {

		return nil, err

	}

	return &StreamDTO{

		Qualities: qualities,
		ProxyURL: baseProxyURL + session.ProxyPath,

		IsHLS: isHLS,

	}, nil

}

func resolveRelativeURL(base *url.URL, ref string) string {

	parsed, err := url.Parse(ref)

	if err != nil {

		return ref

	}

	return base.ResolveReference(parsed).String()

}

func randomToken(n int) (string, error) {

	b := make([]byte, n)

	if _, err := rand.Read(b); err != nil {

		return "", err

	}

	return hex.EncodeToString(b), nil

}

func IsClientDisconnect(err error) bool {

	if err == nil {

		return false

	}

	if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {

		return true

	}

	if errors.Is(err, net.ErrClosed) {

		return true

	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "use of closed network connection")

}

func ForwardMediaResponse(dst http.ResponseWriter, resp *http.Response) error {

	for key, values := range resp.Header {

		if strings.EqualFold(key, "Transfer-Encoding") {

			continue

		}

		for _, value := range values {

			dst.Header().Add(key, value)

		}

	}

	if dst.Header().Get("Accept-Ranges") == "" {

		dst.Header().Set("Accept-Ranges", "bytes")

	}

	dst.Header().Set("Cache-Control", "no-store")

	dst.WriteHeader(resp.StatusCode)

	_, err := io.Copy(dst, resp.Body)

	if IsClientDisconnect(err) {

		return nil

	}

	return err

}

func DetectContentType(urlStr string, header http.Header) string {

	if ct := header.Get("Content-Type"); ct != "" {

		return ct

	}

	lower := strings.ToLower(urlStr)

	if strings.Contains(lower, ".m3u8") || strings.Contains(lower, ".m3u") {

		return "application/vnd.apple.mpegurl"

	}

	if strings.Contains(lower, ".ts") {

		return "video/mp2t"

	}

	if strings.Contains(lower, ".mp4") {

		return "video/mp4"

	}

	return "application/octet-stream"

}

func IsPlaylist(contentType, urlStr string) bool {

	ct := strings.ToLower(contentType)

	if strings.Contains(ct, "mpegurl") || strings.Contains(ct, "m3u") {

		return true

	}

	lower := strings.ToLower(strings.Split(urlStr, "?")[0])

	return strings.HasSuffix(lower, ".m3u8") || strings.HasSuffix(lower, ".m3u")

}

func IsM3U8Body(body []byte) bool {

	trimmed := strings.TrimSpace(string(body))

	return strings.HasPrefix(trimmed, "#EXTM3U")

}

func ProxyError(err error) error {

	if errors.Is(err, mongo.ErrNoDocuments) {

		return fmt.Errorf("stream session expired")

	}

	return err

}
