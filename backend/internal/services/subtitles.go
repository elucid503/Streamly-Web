package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"streamly/internal/captions"
	"streamly/internal/config"

	"golang.org/x/sync/singleflight"
)

const (
	maxSubtitleCacheEntries = 512
	emptySubtitleTTL        = 2 * time.Minute
)

type subtitleCacheEntry struct {

	tracks []SubtitleDTO
	expiry time.Time

}

type SubtitleResolver struct {

	media   *MediaService
	opensubs *captions.OpenSubsClient
	ttl     time.Duration

	cacheMu sync.RWMutex

	movieCache   map[int]subtitleCacheEntry
	episodeCache map[string]subtitleCacheEntry

	group singleflight.Group

}

func NewSubtitleResolver(media *MediaService, opensubs *captions.OpenSubsClient, cfg *config.Config) *SubtitleResolver {

	ttl := cfg.SubtitleCacheTTL

	if ttl <= 0 {

		ttl = 15 * time.Minute

	}

	return &SubtitleResolver{

		media:    media,
		opensubs: opensubs,

		ttl: ttl,

		movieCache:   make(map[int]subtitleCacheEntry),
		episodeCache: make(map[string]subtitleCacheEntry),

	}

}

func (r *SubtitleResolver) MovieTracks(ctx context.Context, id int) []SubtitleDTO {

	if tracks, ok := r.getMovieCached(id); ok {

		return tracks

	}

	result, _, _ := r.group.Do(fmt.Sprintf("movie:%d", id), func() (any, error) {

		tracks := r.resolveMovieTracks(ctx, id)

		r.setMovieCached(id, tracks)

		return tracks, nil

	})

	return cloneSubtitleTracks(result.([]SubtitleDTO))

}

func (r *SubtitleResolver) EpisodeTracks(ctx context.Context, showID, season, episode int) []SubtitleDTO {

	key := episodeCacheKey(showID, season, episode)

	if tracks, ok := r.getEpisodeCached(key); ok {

		return tracks

	}

	result, _, _ := r.group.Do("episode:"+key, func() (any, error) {

		tracks := r.resolveEpisodeTracks(ctx, showID, season, episode)

		r.setEpisodeCached(key, tracks)

		return tracks, nil

	})

	return cloneSubtitleTracks(result.([]SubtitleDTO))

}

func (r *SubtitleResolver) resolveMovieTracks(ctx context.Context, id int) []SubtitleDTO {

	query, err := r.media.MovieCaptionQuery(id)

	return r.opensubsTracks(ctx, query, err)

}

func (r *SubtitleResolver) resolveEpisodeTracks(ctx context.Context, showID, season, episode int) []SubtitleDTO {

	query, err := r.media.EpisodeCaptionQuery(showID, season, episode)

	return r.opensubsTracks(ctx, query, err)

}

func episodeCacheKey(showID, season, episode int) string {

	return fmt.Sprintf("%d:%d:%d", showID, season, episode)

}

func (r *SubtitleResolver) getMovieCached(id int) ([]SubtitleDTO, bool) {

	r.cacheMu.RLock()

	defer r.cacheMu.RUnlock()

	entry, ok := r.movieCache[id]

	if !ok || time.Now().After(entry.expiry) {

		return nil, false

	}

	return cloneSubtitleTracks(entry.tracks), true

}

func (r *SubtitleResolver) setMovieCached(id int, tracks []SubtitleDTO) {

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

func (r *SubtitleResolver) getEpisodeCached(key string) ([]SubtitleDTO, bool) {

	r.cacheMu.RLock()

	defer r.cacheMu.RUnlock()

	entry, ok := r.episodeCache[key]

	if !ok || time.Now().After(entry.expiry) {

		return nil, false

	}

	return cloneSubtitleTracks(entry.tracks), true

}

func (r *SubtitleResolver) setEpisodeCached(key string, tracks []SubtitleDTO) {

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

func cloneSubtitleTracks(tracks []SubtitleDTO) []SubtitleDTO {

	if len(tracks) == 0 {

		return []SubtitleDTO{}

	}

	return append([]SubtitleDTO(nil), tracks...)

}

func (r *SubtitleResolver) opensubsTracks(ctx context.Context, query captions.Query, err error) []SubtitleDTO {

	if err != nil || r.opensubs == nil || !r.opensubs.Configured() {

		return nil

	}

	tracks, err := r.opensubs.ListTracks(ctx, query)

	if err != nil {

		return nil

	}

	out := make([]SubtitleDTO, 0, min(len(tracks), 3))

	langCount := make(map[string]int)

	for _, track := range tracks {

		if len(out) >= 3 {

			break

		}

		content, format, err := r.opensubs.DownloadTrack(ctx, track, query.Season, query.Episode)

		if err != nil {

			continue

		}

		if format == "" {

			format = track.Format

		}

		if format == "zip" {

			continue

		}

		label := friendlyOpenSubsLabel(track, langCount)

		out = append(out, SubtitleDTO{

			ID:       opensubsTrackID(track),
			Label:    label,
			Language: track.Language,
			Format:   format,
			ProxyURL: subtitleDataURI(content, format),
			Source:   "opensubtitles",

		})

	}

	return out

}

func subtitleDataURI(content []byte, format string) string {

	var mimeType string

	switch strings.ToLower(strings.TrimSpace(format)) {

	case "vtt":

		mimeType = "text/vtt;charset=utf-8"

	default:

		mimeType = "text/plain;charset=utf-8"

	}

	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(content)

}

func friendlyLanguageName(code string) string {

	switch strings.ToLower(strings.TrimSpace(code)) {

	case "en", "eng", "english":

		return "English"

	case "es", "spa", "spanish":

		return "Spanish"

	case "fr", "fre", "french":

		return "French"

	case "de", "ger", "german":

		return "German"

	case "it", "ita", "italian":

		return "Italian"

	case "pt", "por", "portuguese":

		return "Portuguese"

	case "und", "":

		return "Unknown"

	default:

		if len(code) == 0 {

			return "Unknown"

		}

		if len(code) == 1 {

			return strings.ToUpper(code)

		}

		return strings.ToUpper(code[:1]) + strings.ToLower(code[1:])

	}

}

func friendlyOpenSubsLabel(track captions.Track, langCount map[string]int) string {

	lang := friendlyLanguageName(track.Language)

	return disambiguateLabel(lang, langCount, track.Hi)

}

func disambiguateLabel(lang string, langCount map[string]int, hearingImpaired bool) string {

	key := lang

	if hearingImpaired {

		key += "+sdh"

	}

	langCount[key]++

	count := langCount[key]

	label := lang

	if hearingImpaired {

		label += " (SDH)"

	}

	if count > 1 {

		label += fmt.Sprintf(" · Option %d", count)

	}

	return label

}

func opensubsTrackID(track captions.Track) string {

	sum := sha256.Sum256([]byte(strings.TrimSpace(track.Path)))

	return "opensubs-" + hex.EncodeToString(sum[:8])

}

func (s *MediaService) MovieCaptionQuery(id int) (captions.Query, error) {

	details, err := s.client.Movie(id).Details()

	if err != nil {

		return captions.Query{}, err

	}

	if details.IMDBId == "" && details.TMDBId <= 0 {

		return captions.Query{}, fmt.Errorf("captions: no metadata ids for movie %d", id)

	}

	return captions.Query{

		IMDBId: details.IMDBId,
		TMDBId: details.TMDBId,

	}, nil

}

func (s *MediaService) EpisodeCaptionQuery(showID, season, episode int) (captions.Query, error) {

	details, err := s.client.Show(showID).Details()

	if err != nil {

		return captions.Query{}, err

	}

	if details.IMDBId == "" && details.TMDBId <= 0 {

		return captions.Query{}, fmt.Errorf("captions: no metadata ids for show %d", showID)

	}

	return captions.Query{

		IMDBId: details.IMDBId,
		TMDBId: details.TMDBId,

		Season:  season,
		Episode: episode,

	}, nil

}
