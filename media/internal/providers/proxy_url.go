package providers

import (
	"net/url"
	"strings"
)

func normalizeProxyURL(raw string) (*url.URL, error) {

	raw = strings.TrimSpace(raw)

	if raw == "" {

		return nil, nil

	}

	if !strings.Contains(raw, "://") {

		raw = "http://" + raw

	}

	raw = strings.Replace(raw, "socks5h://", "socks5://", 1)

	parsed, err := url.Parse(raw)

	if err != nil {

		return nil, err

	}

	if parsed.Scheme == "" {

		parsed.Scheme = "http"

	}

	return parsed, nil

}

func proxyURLForLog(parsed *url.URL) string {

	if parsed == nil {

		return ""

	}

	clone := *parsed

	clone.User = nil

	return clone.String()

}