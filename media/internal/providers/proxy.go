package providers

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func providerHTTPClient() *http.Client {

	return &http.Client{

		Transport: providerTransport(),
		Timeout:   20 * time.Second,

	}

}

func providerTransport() *http.Transport {

	return &http.Transport{

		Proxy: providerProxyFunc(),

		MaxIdleConns:        32,
		MaxIdleConnsPerHost: 8,
		IdleConnTimeout:     90 * time.Second,

	}

}

// VIXSRC_PROXY_URL applies only to server-side Vixsrc resolution (API + embed + master
// playlist fetch). Supports http://, https://, and socks5:// proxies.
func providerProxyFunc() func(*http.Request) (*url.URL, error) {

	raw := strings.TrimSpace(os.Getenv("VIXSRC_PROXY_URL"))

	if raw == "" {

		return http.ProxyFromEnvironment

	}

	parsed, err := normalizeProxyURL(raw)

	if err != nil || parsed == nil {

		streamDebugf("vixsrc proxy url invalid %q: %v", raw, err)

		return http.ProxyFromEnvironment

	}

	streamDebugf("vixsrc resolution using proxy %s", proxyURLForLog(parsed))

	return http.ProxyURL(parsed)

}

func vixsrcServerEnabled() bool {

	value := strings.ToLower(strings.TrimSpace(os.Getenv("VIXSRC_SERVER")))

	switch value {

	case "0", "false", "no", "off":

		return false

	default:

		return true

	}

}