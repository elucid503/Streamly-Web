package services

import (
	"strings"
	"time"

	mediakit "mediakit"
)

type refreshBatch struct {

	searches map[string][]mediakit.SearchHit

}

func (s *MediaService) beginRefreshBatch() *refreshBatch {

	batch := &refreshBatch{searches: make(map[string][]mediakit.SearchHit)}

	s.batchMu.Lock()

	s.batch = batch

	s.batchMu.Unlock()

	return batch

}

func (s *MediaService) endRefreshBatch() {

	s.batchMu.Lock()

	s.batch = nil

	s.batchMu.Unlock()

}

func (s *MediaService) throttleUpstream() {

	s.batchMu.RLock()

	inBatch := s.batch != nil

	s.batchMu.RUnlock()

	if inBatch {

		return

	}

	s.upstreamMu.Lock()

	defer s.upstreamMu.Unlock()

	min := s.cfg.UpstreamMinInterval

	if min <= 0 {

		min = 1500 * time.Millisecond

	}

	if wait := min - time.Since(s.lastUpstream); wait > 0 {

		time.Sleep(wait)

	}

	s.lastUpstream = time.Now()

}

func (s *MediaService) searchUpstream(query string) ([]mediakit.SearchHit, error) {

	query = strings.TrimSpace(query)

	if query == "" {

		return nil, nil

	}

	s.batchMu.RLock()

	batch := s.batch

	s.batchMu.RUnlock()

	if batch != nil {

		if hits, ok := batch.searches[query]; ok {

			return hits, nil

		}

	}

	s.throttleUpstream()

	hits, err := s.client.Search(query)

	if err != nil {

		return nil, err

	}

	if batch != nil {

		batch.searches[query] = hits

	}

	return hits, nil

}

func isRateLimitError(err error) bool {

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

func retryUpstream[T any](attempts int, fn func() (T, error)) (T, error) {

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

		if !isRateLimitError(err) {

			return zero, err

		}

	}

	return zero, last

}
