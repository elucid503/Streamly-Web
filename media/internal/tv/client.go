package tv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL = "https://dami-tv.pro"

	legacyResolvePath = "/papi/tv/resolve/"

	catalogTTL = 120 * time.Minute // 2 hours

)

// Options tunes a Client instance.
type Options struct {
	BaseURL string
}

// Client fetches channel listings and resolves HLS streams.
type Client struct {
	baseURL string
	client  *http.Client

	catalogMu sync.RWMutex
	catalog   *ChannelCatalog
	catalogAt time.Time

	refreshOnce    sync.Once
	enrichmentOnce sync.Once

	metadataMu sync.RWMutex
	metadata   *channelMetadataIndex
	metadataAt time.Time

	sportsMu sync.RWMutex
	sports []SportsEvent
	sportsAt time.Time
	sportsRefreshing bool
}

// New builds a Client with optional overrides.
func New(options Options) *Client {

	baseURL := options.BaseURL

	if baseURL == "" {

		baseURL = os.Getenv("TV_BASE_URL")

	}

	if baseURL == "" {

		baseURL = defaultBaseURL

	}

	client := &Client{

		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{

			Timeout: 30 * time.Second,
		},
	}

	seedEmbeddedCatalog(client)

	return client

}

type streamResolver func(string) (ResolvedStream, error)

// ResolveStream returns the first working HLS playlist for a catalog channel daddyId.
func (c *Client) ResolveStream(daddyID string) (ResolvedStream, error) {

	daddyID = strings.TrimSpace(daddyID)

	if daddyID == "" {

		return ResolvedStream{}, fmt.Errorf("tv: daddyId is required")

	}

	var errs []error

	for _, resolve := range c.streamResolvers() {

		stream, err := resolve(daddyID)

		if err != nil {

			errs = append(errs, err)
			continue

		}

		if !isHLSPlaylistURL(stream.URL) {

			errs = append(errs, fmt.Errorf("tv: not an hls playlist: %s", stream.URL))
			continue

		}

		return stream, nil

	}

	if len(errs) == 0 {

		return ResolvedStream{}, fmt.Errorf("tv: no stream resolvers configured")

	}

	return ResolvedStream{}, errors.Join(errs...)

}

func (c *Client) streamResolvers() []streamResolver {

	resolvers := []streamResolver{

		c.resolveLegacy,
		c.resolveDLHD,
	}

	if api := strings.TrimSpace(os.Getenv("TV_STREAM_API")); api != "" {

		resolvers = append([]streamResolver{c.resolveStreamAPI}, resolvers...)

	}

	return resolvers

}

// resolveLegacy asks the catalog origin for a proxied /papi/tv/resolve/ URL.
func (c *Client) resolveLegacy(daddyID string) (ResolvedStream, error) {

	referer := c.baseURL + "/"
	resolveURL := c.baseURL + legacyResolvePath + url.PathEscape(daddyID)

	return c.fetchResolvedStream(resolveURL, referer, c.baseURL)

}

// resolveStreamAPI resolves through a TV_STREAM_API override endpoint. cfbu247.sbs proxy playlist URLs are rejected because Cloudflare blocks server-side proxying.
func (c *Client) resolveStreamAPI(daddyID string) (ResolvedStream, error) {

	api := strings.TrimSpace(os.Getenv("TV_STREAM_API"))
	resolveURL := joinStreamAPI(api, daddyID)
	referer := streamAPIOrigin(api) + "/"

	stream, err := c.fetchResolvedStream(resolveURL, referer, "")

	if err != nil {

		return ResolvedStream{}, err

	}

	if strings.Contains(stream.URL, "cfbu247.sbs") {

		return ResolvedStream{}, fmt.Errorf("tv: cloudflare proxy playlists are not supported")

	}

	return stream, nil

}

func (c *Client) fetchResolvedStream(resolveURL, referer, origin string) (ResolvedStream, error) {

	response, err := c.get(resolveURL, referer)

	if err != nil {

		return ResolvedStream{}, fmt.Errorf("tv: resolve stream: %w", err)

	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {

		return ResolvedStream{}, fmt.Errorf("tv: read resolve response: %w", err)

	}

	if response.StatusCode != http.StatusOK {

		msg := strings.TrimSpace(string(body))

		if msg == "" {

			msg = response.Status

		}

		return ResolvedStream{}, fmt.Errorf("tv: resolve stream: status %d: %s", response.StatusCode, msg)

	}

	streamURL, err := parseResolveResponse(body)

	if err != nil {

		return ResolvedStream{}, err

	}

	if streamURL == "" {

		return ResolvedStream{}, fmt.Errorf("tv: resolve failed: empty stream path")

	}

	if !strings.HasPrefix(streamURL, "http://") && !strings.HasPrefix(streamURL, "https://") {

		if origin == "" {

			return ResolvedStream{}, fmt.Errorf("tv: resolve failed: relative stream path without origin")

		}

		if !strings.HasPrefix(streamURL, "/") {

			streamURL = "/" + streamURL

		}

		streamURL = origin + streamURL

	}

	return ResolvedStream{

		URL:     streamURL,
		Referer: referer,
	}, nil

}

func isHLSPlaylistURL(raw string) bool {

	lower := strings.ToLower(strings.TrimSpace(raw))
	path := strings.SplitN(lower, "?", 2)[0]

	if strings.HasSuffix(path, ".m3u8") || strings.HasSuffix(path, ".m3u") {

		return true

	}

	return strings.Contains(path, "/papi/tv/playlist/") || strings.Contains(path, "/api/proxy/playlist")

}

// ResolveHLS turns a daddyId into a full proxied m3u8 URL.
func (c *Client) ResolveHLS(daddyID string) (string, error) {

	stream, err := c.ResolveStream(daddyID)

	if err != nil {

		return "", err

	}

	return stream.URL, nil

}

func joinStreamAPI(streamAPI, daddyID string) string {

	streamAPI = strings.TrimSpace(streamAPI)

	if strings.Contains(streamAPI, "?") {

		return streamAPI + url.QueryEscape(daddyID)

	}

	return strings.TrimRight(streamAPI, "/") + "/" + url.PathEscape(daddyID)

}

func streamAPIOrigin(streamAPI string) string {

	parsed, err := url.Parse(strings.TrimSpace(streamAPI))

	if err != nil || parsed.Scheme == "" || parsed.Host == "" {

		return "https://cfbu247.sbs"

	}

	return parsed.Scheme + "://" + parsed.Host

}

func parseResolveResponse(body []byte) (string, error) {

	var tv247 TV247ResolveResult

	if err := json.Unmarshal(body, &tv247); err == nil {

		if tv247.Error != "" {

			return "", fmt.Errorf("tv: resolve failed: %s", tv247.Error)

		}

		if tv247.ProxyPlaylistURL != "" {

			return tv247.ProxyPlaylistURL, nil

		}

	}

	var legacy ResolveResult

	if err := json.Unmarshal(body, &legacy); err != nil {

		return "", fmt.Errorf("tv: decode resolve response: %w", err)

	}

	if !legacy.Success {

		msg := legacy.Error

		if msg == "" {

			msg = string(body)

		}

		return "", fmt.Errorf("tv: resolve failed: %s", msg)

	}

	if legacy.Stream == "" {

		return "", nil

	}

	streamPath := legacy.Stream

	if strings.HasPrefix(streamPath, "http://") || strings.HasPrefix(streamPath, "https://") {

		return streamPath, nil

	}

	if !strings.HasPrefix(streamPath, "/") {

		streamPath = "/" + streamPath

	}

	return streamPath, nil

}

// ResolveChannel resolves HLS for a catalog channel using its daddyId.
func (c *Client) ResolveChannel(channel Channel) (*StreamInfo, error) {

	hls, err := c.ResolveHLS(channel.DaddyID)

	if err != nil {

		return nil, err

	}

	return &StreamInfo{Channel: channel, HLSURL: hls}, nil

}

func (c *Client) cachedCatalog() *ChannelCatalog {

	return c.anyCatalog()

}

func (c *Client) storeCatalog(catalog *ChannelCatalog) {

	c.catalogMu.Lock()
	defer c.catalogMu.Unlock()

	c.catalog = catalog
	c.catalogAt = time.Now()

}

const browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"

func (c *Client) get(rawURL, referer string) (*http.Response, error) {

	request, err := http.NewRequest(http.MethodGet, rawURL, nil)

	if err != nil {

		return nil, err

	}

	if referer == "" {

		referer = c.baseURL + "/"

	}

	request.Header.Set("User-Agent", browserUA)
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Referer", referer)

	return c.client.Do(request)

}
