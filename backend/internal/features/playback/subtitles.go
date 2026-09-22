package playback

import (
	"context"
	"fmt"
	"sync"
	"time"

	"streamly/internal/config"
	"streamly/internal/features/catalog"
	"streamly/internal/features/playback/captions"

	"golang.org/x/sync/singleflight"
)

const (
	maxSubtitleCacheEntries = 512
	emptySubtitleTTL = 2 * time.Minute
)

type subtitleCacheEntry struct {

	tracks []catalog.SubtitleDTO
	expiry time.Time

}

type SubtitleResolver struct {

	media *catalog.MediaService
	subdl *captions.SubDLClient
	opensubs *captions.OpenSubsClient
	ttl time.Duration

	cacheMu sync.RWMutex

	movieCache map[int]subtitleCacheEntry
	episodeCache map[string]subtitleCacheEntry

	group singleflight.Group

}

func NewSubtitleResolver(media *catalog.MediaService, subdl *captions.SubDLClient, opensubs *captions.OpenSubsClient, cfg *config.Config) *SubtitleResolver {

	ttl := cfg.SubtitleCacheTTL

	if ttl <= 0 {

		ttl = 15 * time.Minute

	}

	return &SubtitleResolver{

		media: media,
		subdl: subdl,
		opensubs: opensubs,

		ttl: ttl,

		movieCache: make(map[int]subtitleCacheEntry),
		episodeCache: make(map[string]subtitleCacheEntry),

	}

}

func (r *SubtitleResolver) MovieTracks(ctx context.Context, id int) []catalog.SubtitleDTO {

	if tracks, ok := r.getMovieCached(id); ok {

		return tracks

	}

	result, _, _ := r.group.Do(fmt.Sprintf("movie:%d", id), func() (any, error) {

		tracks := r.resolveMovieTracks(ctx, id)

		r.setMovieCached(id, tracks)

		return tracks, nil

	})

	return cloneSubtitleTracks(result.([]catalog.SubtitleDTO))

}

func (r *SubtitleResolver) EpisodeTracks(ctx context.Context, showID, season, episode int) []catalog.SubtitleDTO {

	key := episodeCacheKey(showID, season, episode)

	if tracks, ok := r.getEpisodeCached(key); ok {

		return tracks

	}

	result, _, _ := r.group.Do("episode:"+key, func() (any, error) {

		tracks := r.resolveEpisodeTracks(ctx, showID, season, episode)

		r.setEpisodeCached(key, tracks)

		return tracks, nil

	})

	return cloneSubtitleTracks(result.([]catalog.SubtitleDTO))

}

func (r *SubtitleResolver) resolveMovieTracks(ctx context.Context, id int) []catalog.SubtitleDTO {

	query, err := r.movieCaptionQuery(id)

	if tracks := r.subdlTracks(ctx, query, err); len(tracks) > 0 {

		return tracks

	}

	return r.opensubsTracks(ctx, query, err)

}

func (r *SubtitleResolver) resolveEpisodeTracks(ctx context.Context, showID, season, episode int) []catalog.SubtitleDTO {

	query, err := r.episodeCaptionQuery(showID, season, episode)

	if tracks := r.subdlTracks(ctx, query, err); len(tracks) > 0 {

		return tracks

	}

	return r.opensubsTracks(ctx, query, err)

}

func (r *SubtitleResolver) subdlTracks(ctx context.Context, query captions.Query, err error) []catalog.SubtitleDTO {

	if err != nil || r.subdl == nil || !r.subdl.Configured() {

		return nil

	}

	tracks, err := r.subdl.ListTracks(ctx, query)

	if err != nil {

		return nil

	}

	out := make([]catalog.SubtitleDTO, 0, min(len(tracks), 3))

	langCount := make(map[string]int)

	for _, track := range tracks {

		if len(out) >= 3 {

			break

		}

		content, format, err := r.subdl.DownloadTrack(ctx, track, query.Season, query.Episode)

		if err != nil {

			continue

		}

		if format == "" {

			format = track.Format

		}

		if format == "zip" {

			continue

		}

		label := friendlySubdlLabel(track, langCount)

		out = append(out, catalog.SubtitleDTO{

			ID: subdlTrackID(track),
			Label: label,
			Language: track.Language,
			Format: format,
			ProxyURL: subtitleDataURI(content, format),
			Source: "subdl",

		})

	}

	return out

}

func (r *SubtitleResolver) opensubsTracks(ctx context.Context, query captions.Query, err error) []catalog.SubtitleDTO {

	if err != nil || r.opensubs == nil || !r.opensubs.Configured() {

		return nil

	}

	tracks, err := r.opensubs.ListTracks(ctx, query)

	if err != nil {

		return nil

	}

	out := make([]catalog.SubtitleDTO, 0, min(len(tracks), 3))

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

		out = append(out, catalog.SubtitleDTO{

			ID: opensubsTrackID(track),
			Label: label,
			Language: track.Language,
			Format: format,
			ProxyURL: subtitleDataURI(content, format),
			Source: "opensubtitles",

		})

	}

	return out

}
