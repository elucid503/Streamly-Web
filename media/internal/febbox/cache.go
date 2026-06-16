package febbox

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	listCacheTTL  = 45 * time.Minute
	linksCacheTTL = 30 * time.Minute
	staleCacheTTL = 24 * time.Hour

	listRequestGap = 250 * time.Millisecond
	linkRequestGap = 2 * time.Second
)

var qualityHeightRe = regexp.MustCompile(`(?i)(\d{3,4})\s*p|2160|4k`)

type listCacheEntry struct {
	files     []File
	fetchedAt time.Time
}

type linksCacheEntry struct {
	qualities []Quality
	fetchedAt time.Time
}

type inflightList struct {
	files []File

	err  error
	done chan struct{}
}

type inflightLinks struct {
	qualities []Quality

	done chan struct{}
	err  error
}

// CachedClient wraps a Client with in-memory caching, request throttling, in-flight deduplication, and stale fallback on rate limits.
type CachedClient struct {
	inner *Client

	mu    sync.RWMutex
	lists map[string]listCacheEntry
	links map[string]linksCacheEntry

	throttleMu sync.Mutex
	lastFetch  time.Time

	listInflight  map[string]*inflightList
	linksInflight map[string]*inflightLinks
	inflightMu    sync.Mutex
}

// NewCached wraps client with defensive caching defaults.
func NewCached(client *Client) *CachedClient {

	return &CachedClient{

		inner: client,

		lists: make(map[string]listCacheEntry),
		links: make(map[string]linksCacheEntry),

		listInflight:  make(map[string]*inflightList),
		linksInflight: make(map[string]*inflightLinks),
	}

}

// ListFiles returns cached folder listings when fresh.
func (c *CachedClient) ListFiles(shareKey string, parentID any, cookie string) ([]File, error) {

	key := listKey(shareKey, parentID)

	if files, ok := c.freshList(key); ok {

		return cloneFiles(files), nil

	}

	call := c.beginListInflight(key)

	if call != nil {

		<-call.done

		if call.err == nil {

			return cloneFiles(call.files), nil

		}

		if stale, ok := c.staleList(key); ok {

			return cloneFiles(stale), nil

		}

		return nil, call.err

	}

	c.throttle(listRequestGap)

	files, err := c.inner.ListFiles(shareKey, parentID, cookie)

	c.finishListInflight(key, files, err)

	if err != nil {

		if stale, ok := c.staleList(key); ok {

			return cloneFiles(stale), nil

		}

		return nil, err

	}

	c.storeList(key, files)

	return cloneFiles(files), nil

}

// GetLinks returns cached quality links when fresh.
func (c *CachedClient) GetLinks(shareKey string, fid any, cookie string) ([]Quality, error) {

	key := linksKey(shareKey, fid)

	if qualities, ok := c.freshLinks(key); ok {

		return cloneQualities(qualities), nil

	}

	call := c.beginLinksInflight(key)

	if call != nil {

		<-call.done

		if call.err == nil {

			return cloneQualities(call.qualities), nil

		}

		if stale, ok := c.staleLinks(key); ok {

			return cloneQualities(stale), nil

		}

		return nil, call.err

	}

	c.throttle(linkRequestGap)

	qualities, err := c.inner.GetLinks(shareKey, fid, cookie)

	c.finishLinksInflight(key, qualities, err)

	if err != nil {

		if stale, ok := c.staleLinks(key); ok {

			return cloneQualities(stale), nil

		}

		return nil, err

	}

	if stale, ok := c.staleLinks(key); ok && isDegradedQualityRefresh(stale, qualities) {

		return cloneQualities(stale), nil

	}

	c.storeLinks(key, qualities)

	return cloneQualities(qualities), nil

}

// GetDownloadURL resolves a direct download link for a shared file.
func (c *CachedClient) GetDownloadURL(shareKey string, fid any, cookie string) (string, error) {

	c.throttle(linkRequestGap)
	return c.inner.GetDownloadURL(shareKey, fid, cookie)

}

func (c *CachedClient) beginListInflight(key string) *inflightList {

	c.inflightMu.Lock()
	defer c.inflightMu.Unlock()

	if call, ok := c.listInflight[key]; ok {

		return call

	}

	call := &inflightList{done: make(chan struct{})}
	c.listInflight[key] = call

	return nil

}

func (c *CachedClient) finishListInflight(key string, files []File, err error) {

	c.inflightMu.Lock()

	call := c.listInflight[key]
	delete(c.listInflight, key)

	c.inflightMu.Unlock()

	if call == nil {

		return

	}

	call.files = files
	call.err = err

	close(call.done)

}

func (c *CachedClient) beginLinksInflight(key string) *inflightLinks {

	c.inflightMu.Lock()
	defer c.inflightMu.Unlock()

	if call, ok := c.linksInflight[key]; ok {

		return call

	}

	call := &inflightLinks{done: make(chan struct{})}
	c.linksInflight[key] = call

	return nil

}

func (c *CachedClient) finishLinksInflight(key string, qualities []Quality, err error) {

	c.inflightMu.Lock()

	call := c.linksInflight[key]
	delete(c.linksInflight, key)

	c.inflightMu.Unlock()

	if call == nil {

		return

	}

	call.qualities = qualities
	call.err = err

	close(call.done)

}

func (c *CachedClient) freshList(key string) ([]File, bool) {

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.lists[key]

	if !ok || time.Since(entry.fetchedAt) > listCacheTTL {

		return nil, false

	}

	return entry.files, true

}

func (c *CachedClient) staleList(key string) ([]File, bool) {

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.lists[key]

	if !ok || time.Since(entry.fetchedAt) > staleCacheTTL {

		return nil, false

	}

	return entry.files, true

}

func (c *CachedClient) storeList(key string, files []File) {

	c.mu.Lock()
	c.lists[key] = listCacheEntry{files: cloneFiles(files), fetchedAt: time.Now()}
	c.mu.Unlock()

}

func (c *CachedClient) freshLinks(key string) ([]Quality, bool) {

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.links[key]

	if !ok || time.Since(entry.fetchedAt) > linksCacheTTL {

		return nil, false

	}

	return entry.qualities, true

}

func (c *CachedClient) staleLinks(key string) ([]Quality, bool) {

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.links[key]

	if !ok || time.Since(entry.fetchedAt) > staleCacheTTL {

		return nil, false

	}

	return entry.qualities, true

}

func (c *CachedClient) storeLinks(key string, qualities []Quality) {

	c.mu.Lock()
	c.links[key] = linksCacheEntry{qualities: cloneQualities(qualities), fetchedAt: time.Now()}
	c.mu.Unlock()

}

func (c *CachedClient) throttle(minGap time.Duration) {

	c.throttleMu.Lock()
	defer c.throttleMu.Unlock()

	if wait := minGap - time.Since(c.lastFetch); wait > 0 {

		time.Sleep(wait)

	}

	c.lastFetch = time.Now()

}

func listKey(shareKey string, parentID any) string {

	return fmt.Sprintf("list:%s:%v", shareKey, parentID)

}

func linksKey(shareKey string, fid any) string {

	return fmt.Sprintf("links:%s:%v", shareKey, fid)

}

func cloneFiles(files []File) []File {

	if len(files) == 0 {

		return []File{}

	}

	return append([]File(nil), files...)

}

func cloneQualities(qualities []Quality) []Quality {

	if len(qualities) == 0 {

		return []Quality{}

	}

	return append([]Quality(nil), qualities...)

}

func isDegradedQualityRefresh(previous, next []Quality) bool {

	return len(previous) > 0 && maxQualityHeight(previous) > 360 && maxQualityHeight(next) <= 360

}

func maxQualityHeight(qualities []Quality) int {

	maxHeight := 0

	for _, q := range qualities {

		if height := parsedQualityHeight(q); height > maxHeight {

			maxHeight = height

		}

	}

	return maxHeight

}

func parsedQualityHeight(q Quality) int {

	label := strings.TrimSpace(q.Quality + " " + q.Name)

	if strings.Contains(strings.ToLower(label), "4k") || strings.Contains(label, "2160") {

		return 2160

	}

	match := qualityHeightRe.FindStringSubmatch(label)

	if len(match) > 1 && match[1] != "" {

		height, _ := strconv.Atoi(match[1])
		return height

	}

	return 0

}

func isRetryableStatus(status string) bool {

	lower := strings.ToLower(status)

	return strings.Contains(lower, "429") || strings.Contains(lower, "502") || strings.Contains(lower, "503") || strings.Contains(lower, "504") || strings.Contains(lower, "525")

}
