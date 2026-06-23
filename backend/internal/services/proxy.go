package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

	"golang.org/x/sync/singleflight"
)

var (
	hlsURIAttr      = regexp.MustCompile(`URI="([^"]+)"`)
	hlsAudioLangRE  = regexp.MustCompile(`(?i)LANGUAGE="([^"]+)"`)
	hlsDefaultRE    = regexp.MustCompile(`(?i)(DEFAULT=)(YES|NO)`)
	hlsAutoselectRE = regexp.MustCompile(`(?i)(AUTOSELECT=)(YES|NO)`)
)

const proxyTokenCacheMax = 4096

type ProxyEntry struct {

	Token string
	TargetURL string
	Referer string
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

	transport := &http.Transport{

		Proxy: http.ProxyFromEnvironment,

		MaxIdleConns: 64,
		MaxIdleConnsPerHost: 16,

		IdleConnTimeout: 90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,

	}

	return &ProxyService{

		ttl: cfg.ProxyTokenTTL,

		tokenByKey: make(map[string]proxyTokenCacheEntry),
		entryByToken: make(map[string]ProxyEntry),

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

func (s *ProxyService) CreateSession(ctx context.Context, targetURL, referer string, isHLS bool) (*ProxySession, error) {

	targetURL = strings.TrimSpace(targetURL)

	if targetURL == "" {

		return nil, errors.New("empty stream url")

	}

	token, err := s.getOrCreateToken(targetURL, referer)

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

	return &entry, nil

}

func (s *ProxyService) Fetch(ctx context.Context, entry *ProxyEntry, incoming http.Header) (*http.Response, error) {

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

func (s *ProxyService) proxyMediaURL(base *url.URL, referer, baseProxyURL, raw string) (string, error) {

	trimmed := strings.TrimSpace(raw)

	if trimmed == "" || strings.HasPrefix(trimmed, "data:") {

		return trimmed, nil

	}

	resolved := resolveRelativeURL(base, trimmed)

	token, err := s.getOrCreateToken(resolved, referer)

	if err != nil {

		return "", err

	}

	return baseProxyURL + "/api/proxy/" + token, nil

}

func (s *ProxyService) RewritePlaylist(body []byte, entry *ProxyEntry, baseProxyURL string) []byte {

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

			if strings.Contains(trimmed, "EXT-X-MEDIA") && strings.Contains(trimmed, "TYPE=SUBTITLES") {

				continue

			}

			rewritten := line

			if strings.Contains(trimmed, "EXT-X-MEDIA") && strings.Contains(trimmed, "TYPE=AUDIO") {

				rewritten = rewriteAudioDefault(rewritten)

			}

			rewritten = hlsURIAttr.ReplaceAllStringFunc(rewritten, func(match string) string {

				parts := hlsURIAttr.FindStringSubmatch(match)

				if len(parts) < 2 {

					return match

				}

				proxyURL, err := s.proxyMediaURL(base, entry.Referer, baseProxyURL, parts[1])

				if err != nil {

					return match

				}

				return `URI="` + proxyURL + `"`

			})

			out = append(out, rewritten)

			continue

		}

		proxyURL, err := s.proxyMediaURL(base, entry.Referer, baseProxyURL, trimmed)

		if err != nil {

			out = append(out, line)

			continue

		}

		out = append(out, proxyURL)

	}

	return []byte(strings.Join(out, "\n"))

}

func (s *ProxyService) getOrCreateToken(targetURL, referer string) (string, error) {

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

		expiresAt := time.Now().Add(s.ttl)

		entry := ProxyEntry{

			Token: token,
			TargetURL: targetURL,
			Referer: referer,
			ExpiresAt: expiresAt,

		}

		s.tokenMu.Lock()

		s.pruneTokenCacheLocked(time.Now())

		s.tokenByKey[key] = proxyTokenCacheEntry{

			token: token,
			expiresAt: expiresAt,

		}

		s.entryByToken[token] = entry

		s.tokenMu.Unlock()

		return token, nil

	})

	if err != nil {

		return "", err

	}

	return result.(string), nil

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

// rewriteAudioDefault sets DEFAULT=YES/AUTOSELECT=YES for English audio tracks
// and DEFAULT=NO for all others, so players default to English automatically.
func rewriteAudioDefault(line string) string {

	langMatch := hlsAudioLangRE.FindStringSubmatch(line)

	if len(langMatch) < 2 {

		return line

	}

	isEnglish := isEnglishAudioLang(langMatch[1])

	want := "NO"

	if isEnglish {

		want = "YES"

	}

	line = hlsDefaultRE.ReplaceAllString(line, "${1}"+want)
	line = hlsAutoselectRE.ReplaceAllString(line, "${1}"+want)

	return line

}

func isEnglishAudioLang(lang string) bool {

	switch strings.ToLower(strings.TrimSpace(lang)) {

	case "en", "eng", "english":

		return true

	}

	return false

}
