package playback

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

func (s *ProxyService) getOrCreateTokenWithHeaders(targetURL, referer string, headers map[string]string) (string, error) {

	key := proxyTokenKey(targetURL, referer, headers)
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
			RequestHeaders: cloneProxyHeaders(headers),
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

		if now.After(entry.ExpiresAt) {

			delete(s.entryByToken, token)

		}

	}

	for key, entry := range s.tokenByKey {

		if now.After(entry.expiresAt) {

			delete(s.tokenByKey, key)

		}

	}

	// Still over cap: drop the tokens furthest from use. ResolveToken renews the
	// ones a live stream is actually pulling, so those sort last.
	for len(s.entryByToken) > proxyTokenCacheMax {

		oldest := ""
		oldestAt := time.Time{}

		for token, entry := range s.entryByToken {

			if oldest == "" || entry.ExpiresAt.Before(oldestAt) {

				oldest = token
				oldestAt = entry.ExpiresAt

			}

		}

		delete(s.entryByToken, oldest)

	}

	for len(s.tokenByKey) > proxyTokenCacheMax {

		oldest := ""
		oldestAt := time.Time{}

		for key, entry := range s.tokenByKey {

			if oldest == "" || entry.expiresAt.Before(oldestAt) {

				oldest = key
				oldestAt = entry.expiresAt

			}

		}

		delete(s.tokenByKey, oldest)

	}

}

func proxyTokenKey(targetURL, referer string, headers map[string]string) string {

	var builder strings.Builder

	builder.WriteString(targetURL)
	builder.WriteString("\x00")
	builder.WriteString(referer)

	for key, value := range headers {

		if value == "" {

			continue

		}

		builder.WriteString("\x00")
		builder.WriteString(strings.ToLower(key))
		builder.WriteString("=")
		builder.WriteString(value)

	}

	sum := sha256.Sum256([]byte(builder.String()))

	return hex.EncodeToString(sum[:])

}

func cloneProxyHeaders(headers map[string]string) map[string]string {

	if len(headers) == 0 {

		return nil

	}

	out := make(map[string]string, len(headers))

	for key, value := range headers {

		if value == "" {

			continue

		}

		out[key] = value

	}

	return out

}

func randomToken(n int) (string, error) {

	b := make([]byte, n)

	if _, err := rand.Read(b); err != nil {

		return "", err

	}

	return hex.EncodeToString(b), nil

}
