package client

import (
	"sync"
	"time"

	"mediakit/internal/catalog/meta"
	"mediakit/internal/live/catalog"
	"mediakit/internal/live/guide"
	"mediakit/internal/live/source"
	livesports "mediakit/internal/live/sports"
	"mediakit/internal/providers/febbox"
	"mediakit/internal/providers/imdb"
	"mediakit/internal/providers/introdb"
	"mediakit/internal/providers/showbox"
	"mediakit/internal/providers/tmdb"

	"golang.org/x/sync/singleflight"
)

type febboxBrowser interface {

	GetMoviePlayFID(imdbID string) (int, error)

	GetConsoleLinks(fid int) ([]febbox.Quality, error)

	ListFiles(shareKey string, parentID any, cookie string) ([]febbox.File, error)

	GetLinks(shareKey string, fid any, cookie string) ([]febbox.Quality, error)

	GetDownloadURL(shareKey string, fid any, cookie string) (string, error)

}

type introFetcher interface {

	GetMedia(query introdb.MediaQuery) (*introdb.MediaRecord, error)

}

const (
	titleDetailsTTL = 2 * time.Hour

	shareKeyTTL = 6 * time.Hour
)

type titleCacheEntry struct {

	details meta.TitleDetails

	expiry time.Time

}

type shareKeyCacheEntry struct {

	key string

	expiry time.Time

}

// Client is the entry point for catalogue search, VOD browsing, and live TV.

type Client struct {

	showbox *showbox.Client

	febbox febboxBrowser

	liveCatalog *catalog.Client

	liveGuide *guide.Client

	liveSports *livesports.Client

	liveSource *source.Resolver

	imdb *imdb.Client

	tmdb *tmdb.Client

	intro introFetcher

	titleMu sync.Mutex

	titleGroup singleflight.Group

	showTitles map[int]titleCacheEntry

	movieTitles map[int]titleCacheEntry

	shareKeyMu sync.Mutex

	shareKeyGroup singleflight.Group

	shareKeys map[string]shareKeyCacheEntry

}

// New builds a Client with optional configuration.

func New(opts ...Option) *Client {

	cfg := &config{}

	for _, opt := range opts {

		opt(cfg)

	}

	applyDefaults(cfg)

	introClient := introdb.New(introdb.Options{APIKey: cfg.introDBKey})

	var intro introFetcher = introClient

	if cfg.cacheIntro {

		intro = introdb.NewCached(introClient)

	}

	febboxClient := febbox.New(febbox.Options{Cookie: cfg.febboxCookie})

	liveCatalog := catalog.New()

	liveGuide := guide.New(liveCatalog)

	// FMHY-evaluated live TV sources: DaddyLive, NTV, Pluto, iptv-org.

	liveSource := source.Default()

	liveSports := livesports.New(liveCatalog, liveSource)

	return &Client{

		showbox: showbox.New(showbox.Options{ChildMode: cfg.childMode}),

		febbox: febbox.NewCached(febboxClient),

		liveCatalog: liveCatalog,

		liveGuide: liveGuide,

		liveSports: liveSports,

		liveSource: liveSource,

		imdb: imdb.New(cfg.tmdbAPIKey),

		tmdb: tmdb.New(cfg.tmdbAPIKey),

		intro: intro,

		showTitles: make(map[int]titleCacheEntry),

		movieTitles: make(map[int]titleCacheEntry),

		shareKeys: make(map[string]shareKeyCacheEntry),

	}

}
