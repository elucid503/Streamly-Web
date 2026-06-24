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
// playlist fetch). Playback segment proxying uses a separate host filter in the backend.
func providerProxyFunc() func(*http.Request) (*url.URL, error) {

	proxyURL := strings.TrimSpace(os.Getenv("VIXSRC_PROXY_URL"))

	if proxyURL == "" {

		return http.ProxyFromEnvironment

	}

	parsed, err := url.Parse(proxyURL)

	if err != nil {

		return http.ProxyFromEnvironment

	}

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