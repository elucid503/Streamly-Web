package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	defaultHTTPTimeout = 20 * time.Second
)

func newHTTPClient(timeout time.Duration) *http.Client {

	if timeout <= 0 {

		timeout = defaultHTTPTimeout

	}

	return &http.Client{

		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {

			if len(via) >= 8 {

				return fmt.Errorf("too many redirects")

			}

			return nil

		},

	}

}

func getText(ctx context.Context, client *http.Client, rawURL string, headers map[string]string) (string, int, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)

	if err != nil {

		return "", 0, err

	}

	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "*/*")

	for k, v := range headers {

		if v != "" {

			req.Header.Set(k, v)

		}

	}

	resp, err := client.Do(req)

	if err != nil {

		return "", 0, err

	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	if err != nil {

		return "", resp.StatusCode, err

	}

	return string(body), resp.StatusCode, nil

}

// verifyPlaylist checks that URL returns an HLS playlist (#EXTM3U).
func verifyPlaylist(ctx context.Context, client *http.Client, rawURL string, headers map[string]string) bool {

	if rawURL == "" {

		return false

	}

	body, status, err := getText(ctx, client, rawURL, headers)

	if err != nil || status < 200 || status >= 300 {

		return false

	}

	return strings.Contains(body, "#EXTM3U")

}

func firstNonEmpty(values ...string) string {

	for _, v := range values {

		if s := strings.TrimSpace(v); s != "" {

			return s

		}

	}

	return ""

}
