package playback

import (
	"fmt"
	"time"

	"streamly/internal/features/catalog"
)

func episodeCacheKey(showID, season, episode int) string {

	return fmt.Sprintf("%d:%d:%d", showID, season, episode)

}

func (r *SubtitleResolver) getMovieCached(id int) ([]catalog.SubtitleDTO, bool) {

	r.cacheMu.RLock()

	defer r.cacheMu.RUnlock()

	entry, ok := r.movieCache[id]

	if !ok || time.Now().After(entry.expiry) {

		return nil, false

	}

	return cloneSubtitleTracks(entry.tracks), true

}

func (r *SubtitleResolver) setMovieCached(id int, tracks []catalog.SubtitleDTO) {

	r.cacheMu.Lock()

	defer r.cacheMu.Unlock()

	ttl := r.ttl

	if len(tracks) == 0 {

		ttl = emptySubtitleTTL

	}

	r.movieCache[id] = subtitleCacheEntry{

		tracks: cloneSubtitleTracks(tracks),
		expiry: time.Now().Add(ttl),

	}

	r.pruneLocked()

}

func (r *SubtitleResolver) getEpisodeCached(key string) ([]catalog.SubtitleDTO, bool) {

	r.cacheMu.RLock()

	defer r.cacheMu.RUnlock()

	entry, ok := r.episodeCache[key]

	if !ok || time.Now().After(entry.expiry) {

		return nil, false

	}

	return cloneSubtitleTracks(entry.tracks), true

}

func (r *SubtitleResolver) setEpisodeCached(key string, tracks []catalog.SubtitleDTO) {

	r.cacheMu.Lock()

	defer r.cacheMu.Unlock()

	ttl := r.ttl

	if len(tracks) == 0 {

		ttl = emptySubtitleTTL

	}

	r.episodeCache[key] = subtitleCacheEntry{

		tracks: cloneSubtitleTracks(tracks),
		expiry: time.Now().Add(ttl),

	}

	r.pruneLocked()

}

func (r *SubtitleResolver) pruneLocked() {

	now := time.Now()

	for id, entry := range r.movieCache {

		if now.After(entry.expiry) || len(r.movieCache) > maxSubtitleCacheEntries {

			delete(r.movieCache, id)

		}

	}

	for key, entry := range r.episodeCache {

		if now.After(entry.expiry) || len(r.episodeCache) > maxSubtitleCacheEntries {

			delete(r.episodeCache, key)

		}

	}

}

func cloneSubtitleTracks(tracks []catalog.SubtitleDTO) []catalog.SubtitleDTO {

	if len(tracks) == 0 {

		return []catalog.SubtitleDTO{}

	}

	return append([]catalog.SubtitleDTO(nil), tracks...)

}
