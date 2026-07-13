package discover

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	mediakit "mediakit"
	"streamly/internal/models"
	"streamly/internal/services/catalog"
)

const (

	sectionLimit = 12
	fetchLimit = 24
	minSectionItems = 2
	tasteCacheTTL = 15 * time.Minute
	globalCacheTTL = time.Hour
	featuredLimit = 6

)

type rawSection struct {

	ID string
	Title string
	Items []mediakit.TMDBItem

}

// Service builds personalized home feeds from TMDB discovery bridged to Showbox.
type Service struct {

	client *mediakit.Client
	bridge *bridgeCache
	fallback Fallback

	mu sync.RWMutex
	movieRaw []rawSection
	showRaw []rawSection
	globalAt time.Time

	tasteMu sync.Mutex
	taste map[string]tasteCacheEntry

	cancel context.CancelFunc

}

// Fallback supplies Showbox catalog data for local matching and soft fallbacks.
type Fallback interface {

	CatalogTrendingHits(kind mediakit.MediaKind, limit int) []catalog.SearchResultDTO
	CatalogIndex() []catalog.SearchResultDTO
	SearchTitles(query string) []catalog.SearchResultDTO

}

type tasteCacheEntry struct {

	profile TasteProfile
	expiry time.Time

}

// New builds a discover Service. bridgeFile is the disk path for TMDB→Showbox mappings.
func New(client *mediakit.Client, bridgeFile string, fallback Fallback) *Service {

	if bridgeFile == "" {

		bridgeFile = filepath.Join("data", "bridge.cache.json")

	}

	return &Service{

		client: client,
		bridge: newBridgeCache(bridgeFile),
		fallback: fallback,
		taste: make(map[string]tasteCacheEntry),

	}

}

// Start refreshes TMDB list metadata on an interval (Showbox bridging is on-click).
func (s *Service) Start(ctx context.Context, ttl time.Duration) {

	if ttl <= 0 {

		ttl = globalCacheTTL

	}

	child, cancel := context.WithCancel(ctx)

	s.cancel = cancel

	go func() {

		s.refreshGlobal()

		ticker := time.NewTicker(ttl)

		defer ticker.Stop()

		for {

			select {

			case <-child.Done():

				s.bridge.flush()
				return

			case <-ticker.C:

				s.refreshGlobal()
				s.bridge.flush()

			}

		}

	}()

}

// Stop cancels the refresh loop.
func (s *Service) Stop() {

	if s.cancel != nil {

		s.cancel()

	}

	s.bridge.flush()

}

// Feed builds a home feed. Items are TMDB-backed; Showbox ids are attached only
// when already cached from a prior click-time resolve.
func (s *Service) Feed(kind string, history []models.WatchHistoryItem, favorites []models.FavoriteItem) HomeFeed {

	tmdbKind := mediakit.TMDBMovie

	if kind == "show" {

		tmdbKind = mediakit.TMDBTV

	}

	s.ensureGlobal()

	taste := s.computeTaste(kind, history, favorites)

	log.Printf("discover: feed %s taste hasData=%v genres=%d seed=%q(%d) history=%d favorites=%d",
		kind, taste.HasData, len(taste.TopGenres), taste.SeedTitle, taste.SeedTMDBID, len(history), len(favorites))

	seen := make(map[int]struct{})
	sections := make([]FeedSection, 0, 10)

	if taste.HasData {

		if taste.SeedTMDBID > 0 {

			recs, err := s.client.TMDB().Recommendations(tmdbKind, taste.SeedTMDBID, fetchLimit)

			if err != nil || len(recs) == 0 {

				recs, err = s.client.TMDB().Similar(tmdbKind, taste.SeedTMDBID, fetchLimit)

			}

			if err == nil {

				items := s.tmdbSection(recs, sectionLimit, seen, "Because you watched "+taste.SeedTitle)

				if len(items) >= minSectionItems {

					sections = append(sections, FeedSection{

						ID: "because-you-watched",
						Title: "Because You Watched " + taste.SeedTitle,
						Kind: "personalized",
						Items: items,

					})

				}

			}

		}

		if len(taste.TopGenres) > 0 {

			genreIDs := make([]int, 0, 2)

			for i, g := range taste.TopGenres {

				if i >= 2 {

					break

				}

				genreIDs = append(genreIDs, g.ID)

			}

			picks, err := s.client.TMDB().Discover(tmdbKind, genreIDs, 6.0, 40, fetchLimit)

			if err == nil {

				items := s.tmdbSection(picks, sectionLimit, seen, "Top pick for you")

				if len(items) >= minSectionItems {

					sections = append(sections, FeedSection{

						ID: "top-picks",
						Title: "Top Picks For You",
						Kind: "personalized",
						Items: items,

					})

				}

			}

			for i, g := range taste.TopGenres {

				if i >= 2 {

					break

				}

				genreItems, err := s.client.TMDB().Discover(tmdbKind, []int{g.ID}, 5.5, 20, fetchLimit)

				if err != nil {

					continue

				}

				items := s.tmdbSection(genreItems, sectionLimit, seen, "")

				if len(items) < minSectionItems {

					continue

				}

				sections = append(sections, FeedSection{

					ID: "genre-" + strconv.Itoa(g.ID),
					Title: g.Name + " For You",
					Kind: "personalized",
					Items: items,

				})

			}

		}

	}

	for _, raw := range s.rawSections(kind) {

		items := s.tmdbSection(raw.Items, sectionLimit, seen, "")

		if len(items) == 0 {

			continue

		}

		sections = append(sections, FeedSection{

			ID: raw.ID,
			Title: raw.Title,
			Kind: "editorial",
			Items: items,

		})

	}

	if len(sections) == 0 {

		sections = s.fallbackSections(mediaKindOf(tmdbKind))

	}

	featured := collectFeatured(sections, featuredLimit)

	return HomeFeed{

		Featured: featured,
		Sections: sections,
		RefreshedAt: time.Now().UTC(),

	}

}

// Resolve maps a TMDB title to a Showbox id (click-time bridge).
func (s *Service) Resolve(kind string, tmdbID int, title string, year int) (*ResolveResult, error) {

	if tmdbID <= 0 {

		return nil, fmt.Errorf("tmdb id required")

	}

	tmdbKind := mediakit.TMDBMovie

	if kind == "show" {

		tmdbKind = mediakit.TMDBTV

	}

	if cached, ok := s.bridge.get(kind, tmdbID); ok && !cached.Miss && cached.ShowboxID > 0 {

		return &ResolveResult{

			ID: cached.ShowboxID,
			TMDBID: tmdbID,
			Kind: kind,
			Title: coalesce(cached.Title, title),
			Year: cached.Year,
			Poster: cached.Poster,

		}, nil

	}

	item := mediakit.TMDBItem{

		ID: tmdbID,
		Kind: tmdbKind,
		Title: title,
		Year: year,

	}

	// Prefer live TMDB metadata when title wasn't provided by the client.
	if title == "" || year == 0 {

		if details, err := s.client.TMDB().Details(tmdbKind, tmdbID); err == nil {

			if item.Title == "" {

				item.Title = details.Title

			}

			if item.Year == 0 {

				item.Year = details.Year

			}

			item.Poster = details.Poster
			item.Backdrop = details.Backdrop
			item.Overview = details.Overview
			item.VoteAverage = details.VoteAverage
			item.GenreIDs = details.GenreIDs

		}

	}

	feed, ok := s.resolve(item)

	if !ok || feed.ID <= 0 {

		cands := s.searchCandidates(item.Title, mediaKindOf(tmdbKind))
		log.Printf("discover: resolve miss %s %q tmdb=%d candidates=%d", kind, item.Title, tmdbID, len(cands))

		for i, c := range cands {

			if i >= 5 {

				break

			}

			log.Printf("discover:   candidate[%d] id=%d title=%q year=%d", i, c.ID, c.Title, c.Year)

		}

		return nil, fmt.Errorf("could not find a streamable match for %q", coalesce(item.Title, fmt.Sprintf("tmdb:%d", tmdbID)))

	}

	s.bridge.flush()

	return &ResolveResult{

		ID: feed.ID,
		TMDBID: tmdbID,
		Kind: kind,
		Title: feed.Title,
		Year: feed.Year,
		Poster: feed.Poster,

	}, nil

}

func (s *Service) tmdbSection(items []mediakit.TMDBItem, limit int, seen map[int]struct{}, reason string) []FeedItem {

	out := make([]FeedItem, 0, limit)

	for _, item := range items {

		if item.ID <= 0 || strings.TrimSpace(item.Title) == "" {

			continue

		}

		if _, ok := seen[item.ID]; ok {

			continue

		}

		seen[item.ID] = struct{}{}

		feed := s.tmdbToFeed(item)

		if reason != "" {

			feed.MatchReason = reason

		}

		out = append(out, feed)

		if len(out) >= limit {

			break

		}

	}

	return out

}

func (s *Service) tmdbToFeed(item mediakit.TMDBItem) FeedItem {

	kind := kindName(item.Kind)

	feed := FeedItem{

		TMDBID: item.ID,
		Kind: kind,

		Title: item.Title,
		Year: item.Year,

		Poster: item.Poster,
		Backdrop: item.Backdrop,
		Description: item.Overview,
		Rating: formatRating(item.VoteAverage),

		Genres: mediakit.TMDBGenreNames(item.GenreIDs),
		Runtime: item.Runtime,

	}

	// Attach Showbox id only when a prior click already resolved it.
	if cached, ok := s.bridge.get(kind, item.ID); ok && !cached.Miss && cached.ShowboxID > 0 {

		feed.ID = cached.ShowboxID

		if feed.Poster == "" {

			feed.Poster = cached.Poster

		}

	}

	return feed

}

func collectFeatured(sections []FeedSection, limit int) []FeedItem {

	out := make([]FeedItem, 0, limit)
	seen := make(map[int]struct{})

	add := func(item FeedItem, fallbackReason string) {

		key := item.TMDBID

		if key <= 0 {

			key = item.ID

		}

		if item.Backdrop == "" || key <= 0 {

			return

		}

		if _, ok := seen[key]; ok {

			return

		}

		if item.MatchReason == "" {

			item.MatchReason = fallbackReason

		}

		seen[key] = struct{}{}
		out = append(out, item)

	}

	for _, section := range sections {

		if section.Kind != "personalized" {

			continue

		}

		for _, item := range section.Items {

			add(item, "Top Pick For You")

			if len(out) >= limit {

				return out

			}

		}

	}

	for _, section := range sections {

		for _, item := range section.Items {

			add(item, "Featured")

			if len(out) >= limit {

				return out

			}

		}

	}

	return out

}

func (s *Service) ensureGlobal() {

	s.mu.RLock()

	fresh := time.Since(s.globalAt) < globalCacheTTL && (len(s.movieRaw) > 0 || len(s.showRaw) > 0)

	s.mu.RUnlock()

	if fresh {

		return

	}

	s.refreshGlobal()

}

func (s *Service) rawSections(kind string) []rawSection {

	s.mu.RLock()
	defer s.mu.RUnlock()

	if kind == "show" {

		return append([]rawSection(nil), s.showRaw...)

	}

	return append([]rawSection(nil), s.movieRaw...)

}

func (s *Service) refreshGlobal() {

	tmdb := s.client.TMDB()

	if tmdb == nil || !tmdb.Enabled() {

		log.Println("discover: TMDB disabled; raw lists empty")
		s.setRaw(nil, nil)
		return

	}

	movieRaw := s.fetchRaw(mediakit.TMDBMovie)
	showRaw := s.fetchRaw(mediakit.TMDBTV)

	s.setRaw(movieRaw, showRaw)

	log.Printf("discover: refreshed TMDB lists movies=%d shows=%d (resolve on click)", len(movieRaw), len(showRaw))

}

func (s *Service) setRaw(movies, shows []rawSection) {

	s.mu.Lock()
	s.movieRaw = movies
	s.showRaw = shows
	s.globalAt = time.Now()
	s.mu.Unlock()

}

func (s *Service) fallbackSections(kind mediakit.MediaKind) []FeedSection {

	if s.fallback == nil {

		return nil

	}

	hits := s.fallback.CatalogTrendingHits(kind, sectionLimit)

	if len(hits) == 0 {

		return nil

	}

	items := make([]FeedItem, 0, len(hits))

	for _, hit := range hits {

		items = append(items, FeedItem{

			ID: hit.ID,
			Kind: hit.Kind,
			Title: hit.Title,
			Year: hit.Year,
			Poster: hit.Poster,
			Description: hit.Description,
			Rating: hit.Rating,

		})

	}

	return []FeedSection{{

		ID: "trending",
		Title: "Trending Now",
		Kind: "editorial",
		Items: items,

	}}

}

func (s *Service) fetchRaw(kind mediakit.TMDBKind) []rawSection {

	tmdb := s.client.TMDB()
	sections := make([]rawSection, 0, 4)

	add := func(id, title string, items []mediakit.TMDBItem, err error) {

		if err != nil || len(items) == 0 {

			return

		}

		sections = append(sections, rawSection{ID: id, Title: title, Items: items})

	}

	trending, err := tmdb.Trending(kind, fetchLimit)
	add("trending", "Trending Now", trending, err)

	popular, err := tmdb.Popular(kind, fetchLimit)
	add("popular", "Popular", popular, err)

	if kind == mediakit.TMDBMovie {

		now, err := tmdb.NowPlaying(fetchLimit)
		add("new-releases", "New In Theaters", now, err)

		acclaimed, err := tmdb.Discover(kind, nil, 7.5, 200, fetchLimit)
		add("acclaimed", "Critically Acclaimed", acclaimed, err)

	} else {

		airing, err := tmdb.OnTheAir(fetchLimit)
		add("new-episodes", "On The Air", airing, err)

		acclaimed, err := tmdb.Discover(kind, nil, 7.5, 200, fetchLimit)
		add("acclaimed", "Critically Acclaimed", acclaimed, err)

	}

	return sections

}
