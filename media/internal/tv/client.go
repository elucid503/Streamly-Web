package tv

import (
	"encoding/json"
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
	defaultStreamAPI = "https://chat.cfbu247.sbs/api/resolve-dlstream/"

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
	client *http.Client

	catalogMu sync.RWMutex
	catalog *ChannelCatalog
	catalogAt time.Time

	refreshOnce sync.Once

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

// ResolveStream turns a daddyId into a direct CDN playlist URL when possible.
func (c *Client) ResolveStream(daddyID string) (ResolvedStream, error) {

	daddyID = strings.TrimSpace(daddyID)

	if daddyID == "" {

		return ResolvedStream{}, fmt.Errorf("tv: daddyId is required")

	}

	// dlhd is tried first: it returns the embed page as the Referer, which the segment CDN

	if stream, err := c.resolveDLHD(daddyID); err == nil && stream.URL != "" {

		return stream, nil

	}

	return c.resolveTV247(daddyID)

}

// ResolveHLS turns a daddyId into a full proxied m3u8 URL.
func (c *Client) ResolveHLS(daddyID string) (string, error) {

	stream, err := c.ResolveStream(daddyID)

	if err != nil {

		return "", err

	}

	return stream.URL, nil

}

func (c *Client) resolveTV247(daddyID string) (ResolvedStream, error) {

	resolveURL, referer, err := c.resolveEndpoint(daddyID)

	if err != nil {

		return ResolvedStream{}, err

	}

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

		if !strings.HasPrefix(streamURL, "/") {

			streamURL = "/" + streamURL

		}

		streamURL = c.baseURL + streamURL

	}

	return ResolvedStream{

		URL: streamURL,
		Referer: referer,

	}, nil

}

func (c *Client) resolveEndpoint(daddyID string) (resolveURL, referer string, err error) {

	streamAPI := c.streamAPI()

	if streamAPI != "" {

		resolveURL = joinStreamAPI(streamAPI, daddyID)
		referer = streamAPIOrigin(streamAPI) + "/"

		return resolveURL, referer, nil

	}

	resolveURL = c.baseURL + legacyResolvePath + url.PathEscape(daddyID)
	referer = c.baseURL + "/"

	return resolveURL, referer, nil

}

func (c *Client) streamAPI() string {

	if cached := c.cachedCatalog(); cached != nil && strings.TrimSpace(cached.StreamAPI) != "" {

		return strings.TrimSpace(cached.StreamAPI)

	}

	if override := strings.TrimSpace(os.Getenv("TV_STREAM_API")); override != "" {

		return override

	}

	return defaultStreamAPI

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

		referer = streamAPIOrigin(c.streamAPI()) + "/"

	}

	request.Header.Set("User-Agent", browserUA)
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Referer", referer)

	return c.client.Do(request)

}
