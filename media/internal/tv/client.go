package tv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL = "https://ntv.cx"

	catalogTTL            = 60 * time.Minute
	catalogRefreshTimeout = 30 * time.Second

	browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
)

// Options tunes a Client instance.
type Options struct {

	BaseURL string

}

// Client fetches the ntv.cx channel listing and resolves HLS streams.
type Client struct {

	baseURL    string
	httpClient *http.Client

	mu      sync.RWMutex
	catalog *ChannelCatalog

	refreshOnce    sync.Once
	enrichmentOnce sync.Once

	metadataMu sync.RWMutex
	metadata   *channelMetadataIndex
	metadataAt time.Time

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

	return &Client{

		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{

			Timeout: 30 * time.Second,
		},
	}

}

func (c *Client) get(rawURL string) (*http.Response, error) {

	request, err := http.NewRequest(http.MethodGet, rawURL, nil)

	if err != nil {

		return nil, err

	}

	request.Header.Set("User-Agent", browserUA)
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")

	return c.httpClient.Do(request)

}

// RefreshCatalog fetches the current channel list from ntv.cx and caches it.
func (c *Client) RefreshCatalog() (*ChannelCatalog, error) {

	response, err := c.get(c.baseURL + "/api/get-channels")

	if err != nil {

		return nil, fmt.Errorf("tv: fetch channels: %w", err)

	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {

		return nil, fmt.Errorf("tv: read channels response: %w", err)

	}

	if response.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("tv: fetch channels: status %d", response.StatusCode)

	}

	var parsed getChannelsResponse

	if err := json.Unmarshal(body, &parsed); err != nil {

		return nil, fmt.Errorf("tv: decode channels response: %w", err)

	}

	if !parsed.Success {

		return nil, fmt.Errorf("tv: channels response reported failure")

	}

	channels := make([]Channel, 0, len(parsed.Channels))

	for _, ch := range parsed.Channels {

		if ch.Server == reliableServer {

			channels = append(channels, ch)

		}

	}

	catalog := &ChannelCatalog{

		Channels:  channels,
		FetchedAt: time.Now(),
	}

	catalog, _ = c.enrichCatalogWithCachedMetadata(catalog)

	c.store(catalog)

	return catalog, nil

}

// ListChannels returns the in-memory channel catalog without performing network I/O.
func (c *Client) ListChannels() (*ChannelCatalog, error) {

	if cached := c.cached(); cached != nil {

		return cached, nil

	}

	return nil, fmt.Errorf("tv: catalog unavailable")

}

func (c *Client) store(catalog *ChannelCatalog) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.catalog = catalog

}

func (c *Client) cached() *ChannelCatalog {

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.catalog == nil {

		return nil

	}

	clone := *c.catalog
	clone.Channels = append([]Channel(nil), c.catalog.Channels...)

	return &clone

}
