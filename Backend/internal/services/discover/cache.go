package discover

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// bridgeEntry maps a TMDB id to a Showbox id with enriched display fields.
type bridgeEntry struct {

	TMDBID int `json:"tmdbId"`
	ShowboxID int `json:"showboxId"`
	Kind string `json:"kind"`

	Title string `json:"title"`
	Year int `json:"year"`
	Poster string `json:"poster"`
	Backdrop string `json:"backdrop,omitempty"`
	Description string `json:"description"`
	Rating string `json:"rating"`
	Genres []string `json:"genres,omitempty"`
	Runtime int `json:"runtime,omitempty"`

	ResolvedAt time.Time `json:"resolvedAt"`
	Miss bool `json:"miss,omitempty"`

}

type bridgeCache struct {

	mu sync.RWMutex
	entries map[string]bridgeEntry
	path string
	dirty bool

}

func newBridgeCache(path string) *bridgeCache {

	c := &bridgeCache{

		entries: make(map[string]bridgeEntry),
		path: path,

	}

	// Drop the previous cache file — the verified-only bridge algorithm
	// invalidates any mappings accepted under the old scorer.
	if path != "" {

		_ = os.Remove(path)

	}

	return c

}

func bridgeKey(kind string, tmdbID int) string {

	return kind + ":" + strconv.Itoa(tmdbID)

}

func (c *bridgeCache) get(kind string, tmdbID int) (bridgeEntry, bool) {

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[bridgeKey(kind, tmdbID)]

	if !ok {

		return bridgeEntry{}, false

	}

	ttl := 7 * 24 * time.Hour

	if entry.Miss {

		// Short miss TTL so bad searches can be retried after algorithm fixes.
		ttl = 30 * time.Minute

	}

	if time.Since(entry.ResolvedAt) > ttl {

		return bridgeEntry{}, false

	}

	return entry, true

}

// clearMisses drops cached negative lookups so a new algorithm can retry.
func (c *bridgeCache) clearMisses() {

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.entries {

		if entry.Miss || entry.ShowboxID <= 0 {

			delete(c.entries, key)
			c.dirty = true

		}

	}

}

func (c *bridgeCache) put(entry bridgeEntry) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[bridgeKey(entry.Kind, entry.TMDBID)] = entry
	c.dirty = true

}

func (c *bridgeCache) load() {

	if c.path == "" {

		return

	}

	data, err := os.ReadFile(c.path)

	if err != nil {

		return

	}

	var raw map[string]bridgeEntry

	if err := json.Unmarshal(data, &raw); err != nil {

		log.Printf("discover: bridge cache load failed: %v", err)
		return

	}

	c.mu.Lock()
	c.entries = raw
	c.mu.Unlock()

}

func (c *bridgeCache) flush() {

	c.mu.Lock()

	if !c.dirty || c.path == "" {

		c.mu.Unlock()
		return

	}

	snapshot := make(map[string]bridgeEntry, len(c.entries))

	for k, v := range c.entries {

		snapshot[k] = v

	}

	c.dirty = false
	c.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {

		log.Printf("discover: bridge cache mkdir: %v", err)
		return

	}

	data, err := json.Marshal(snapshot)

	if err != nil {

		return

	}

	tmp := c.path + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {

		log.Printf("discover: bridge cache write: %v", err)
		return

	}

	_ = os.Rename(tmp, c.path)

}

func normalizeTitle(s string) string {

	var b strings.Builder

	b.Grow(len(s))

	for _, r := range strings.ToLower(strings.TrimSpace(s)) {

		if unicode.IsLetter(r) || unicode.IsDigit(r) {

			b.WriteRune(r)

		}

	}

	return b.String()

}
