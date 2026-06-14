package upstream

import (
	"strings"
	"sync"
	"time"

	mediakit "mediakit"
)

type refreshBatch struct {

	searches map[string][]mediakit.SearchHit

}

// Throttle rate-limits upstream API calls and deduplicates search queries
// within a single batch refresh.
type Throttle struct {

	minInterval time.Duration

	mu sync.Mutex
	last time.Time

	batchMu sync.RWMutex
	batch *refreshBatch

}

// New builds a Throttle. If minInterval is zero, 1500ms is used.
func New(minInterval time.Duration) *Throttle {

	if minInterval <= 0 {

		minInterval = 1500 * time.Millisecond

	}

	return &Throttle{minInterval: minInterval}

}

// Begin starts a batch refresh, enabling query deduplication across calls.
func (t *Throttle) Begin() {

	batch := &refreshBatch{searches: make(map[string][]mediakit.SearchHit)}

	t.batchMu.Lock()

	t.batch = batch

	t.batchMu.Unlock()

}

// End clears the active batch.
func (t *Throttle) End() {

	t.batchMu.Lock()

	t.batch = nil

	t.batchMu.Unlock()

}

// Before sleeps if needed to respect the minimum interval between upstream calls.
// It is a no-op during an active batch refresh.
func (t *Throttle) Before() {

	t.batchMu.RLock()

	inBatch := t.batch != nil

	t.batchMu.RUnlock()

	if inBatch {

		return

	}

	t.mu.Lock()

	defer t.mu.Unlock()

	if wait := t.minInterval - time.Since(t.last); wait > 0 {

		time.Sleep(wait)

	}

	t.last = time.Now()

}

// Searcher is any type that can execute a keyword search.
type Searcher interface {

	Search(query string) ([]mediakit.SearchHit, error)

}

// Search throttles then queries the upstream, caching results within an active batch.
func (t *Throttle) Search(s Searcher, query string) ([]mediakit.SearchHit, error) {

	query = strings.TrimSpace(query)

	if query == "" {

		return nil, nil

	}

	t.batchMu.RLock()

	batch := t.batch

	t.batchMu.RUnlock()

	if batch != nil {

		if hits, ok := batch.searches[query]; ok {

			return hits, nil

		}

	}

	t.Before()

	hits, err := s.Search(query)

	if err != nil {

		return nil, err

	}

	if batch != nil {

		batch.searches[query] = hits

	}

	return hits, nil

}

// Retry calls fn up to attempts times, retrying only on rate-limit errors.
func Retry[T any](attempts int, fn func() (T, error)) (T, error) {

	var zero T

	var last error

	backoff := 5 * time.Second

	for i := 0; i < attempts; i++ {

		if i > 0 {

			time.Sleep(backoff)

			backoff *= 2

		}

		result, err := fn()

		if err == nil {

			return result, nil

		}

		last = err

		if !IsRateLimitError(err) {

			return zero, err

		}

	}

	return zero, last

}

// IsRateLimitError reports whether an error looks like an upstream rate limit.
func IsRateLimitError(err error) bool {

	if err == nil {

		return false

	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "1015") ||
		strings.Contains(msg, "febbox:") ||
		strings.Contains(msg, "525")

}
