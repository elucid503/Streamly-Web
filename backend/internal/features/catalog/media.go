package catalog

import (
	"context"
	"sync"
	"time"

	"mediakit"
	"streamly/internal/config"
	catalogindex "streamly/internal/features/catalog/index"
	"streamly/internal/features/catalog/search"
	"streamly/internal/features/catalog/vod"
	stream "streamly/internal/features/playback/cache"

	"golang.org/x/sync/singleflight"
)

const (
	titleDetailsTTL = 6 * time.Hour
	titleDetailsMaxEntries = 1024
)

// MediaService is the central media orchestrator.
type MediaService struct {

	client *mediakit.Client
	cfg *config.Config

	catalog *catalogindex.Cache

	search *search.Cache
	stream *stream.Cache

	vod *vod.Cache

	detailsMu sync.RWMutex
	detailsGroup singleflight.Group

	movieDetails map[int]titleDetailsCacheEntry
	showDetails map[int]titleDetailsCacheEntry

}

func NewMediaService(cfg *config.Config) *MediaService {

	client := mediakit.New(

		mediakit.WithFebboxCookie(cfg.FebboxCookie),
		mediakit.WithIntroDBKey(cfg.IntroDBKey),
		mediakit.WithTMDBAPIKey(cfg.TMDBAPIKey),
		mediakit.WithChildMode(cfg.ChildMode),
		mediakit.WithIntroCache(true),
	)

	client.Warmup()

	cat := catalogindex.New(client, cfg.CatalogCacheTTL, cfg.CatalogCacheFile)

	return &MediaService{

		client: client,
		cfg: cfg,

		catalog: cat,

		search: search.New(client, cat.Snapshot, cfg.CatalogCacheTTL),
		stream: stream.New(client),

		vod: vod.New(client),

		movieDetails: make(map[int]titleDetailsCacheEntry),
		showDetails: make(map[int]titleDetailsCacheEntry),

	}

}

func (s *MediaService) Client() *mediakit.Client {

	return s.client

}

func (s *MediaService) StartCatalogCache(ctx context.Context) {

	s.catalog.Start(ctx)

}

func (s *MediaService) StopCatalogCache() {

	s.catalog.Stop()

}

func (s *MediaService) TrendingHits(kind mediakit.MediaKind, limit int) ([]SearchResultDTO, error) {

	return s.catalog.TrendingHits(kind, limit), nil

}

func (s *MediaService) CatalogTrendingHits(kind mediakit.MediaKind, limit int) []SearchResultDTO {

	return s.catalog.TrendingHits(kind, limit)

}

// CatalogIndex returns the in-memory Showbox catalog for local title matching.
func (s *MediaService) CatalogIndex() []SearchResultDTO {

	return s.catalog.Snapshot().SearchIndex()

}

// SearchTitles runs the same search path as the UI search box.
func (s *MediaService) SearchTitles(query string) []SearchResultDTO {

	results, err := s.search.Search(query)

	if err != nil || results == nil {

		return nil

	}

	return results

}

func (s *MediaService) Trending(kind mediakit.MediaKind, limit int) ([]string, error) {

	return s.client.Trending(kind, limit)

}

func (s *MediaService) Categories(kind mediakit.MediaKind) ([]CategoryDTO, error) {

	return s.catalog.Categories(kind), nil

}

func (s *MediaService) CategoryTitles(kind mediakit.MediaKind, categoryID string, page, limit int) ([]SearchResultDTO, error) {

	return s.catalog.CategoryTitles(kind, categoryID, page, limit), nil

}

func (s *MediaService) Search(query string) ([]SearchResultDTO, error) {

	return s.search.Search(query)

}
