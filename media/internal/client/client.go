package client

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"mediakit/internal/discover"
	"mediakit/internal/febbox"
	"mediakit/internal/imdb"
	"mediakit/internal/introdb"
	"mediakit/internal/live/catalog"
	"mediakit/internal/live/guide"
	"mediakit/internal/live/source"
	livesports "mediakit/internal/live/sports"
	"mediakit/internal/meta"

	"mediakit/internal/showbox"
	"mediakit/internal/tmdb"
	"mediakit/internal/vod"

	"golang.org/x/sync/singleflight"
)

// Option configures a Client.
type Option func(*config)

type config struct {
	childMode string

	febboxCookie string
	introDBKey   string
	tmdbAPIKey   string

	cacheIntro bool
}

// WithChildMode sets the Showbox child-mode flag.
func WithChildMode(mode string) Option {

	return func(c *config) { c.childMode = mode }

}

// WithFebboxCookie sets the Febbox `ui` auth cookie required for quality links.
func WithFebboxCookie(cookie string) Option {

	return func(c *config) { c.febboxCookie = cookie }

}

// WithIntroDBKey sets an optional TheIntroDB API key.
func WithIntroDBKey(key string) Option {

	return func(c *config) { c.introDBKey = key }

}

// WithTMDBAPIKey sets the TMDB v3 API key for episode and title metadata.
func WithTMDBAPIKey(key string) Option {

	return func(c *config) { c.tmdbAPIKey = key }

}

// WithTVBaseURL is retained for API compatibility but no longer used.
// Live catalog is metadata-only and does not depend on a stream provider origin.
func WithTVBaseURL(baseURL string) Option {

	return func(c *config) {}

}

// WithIntroCache enables a 6-hour in-memory cache for TheIntroDB lookups.
func WithIntroCache(enabled bool) Option {

	return func(c *config) { c.cacheIntro = enabled }

}

func applyDefaults(c *config) {

	if c.childMode == "" {

		c.childMode = os.Getenv("CHILD_MODE")

	}

	if c.childMode == "" {

		c.childMode = "0"

	}

	if c.febboxCookie == "" {

		c.febboxCookie = os.Getenv("FEBBOX_UI_COOKIE")

	}

	if c.introDBKey == "" {

		c.introDBKey = os.Getenv("INTRODB_API_KEY")

	}

	if c.tmdbAPIKey == "" {

		c.tmdbAPIKey = os.Getenv("TMDB_API_KEY")

	}

}

type febboxBrowser interface {
	GetMoviePlayFID(imdbID string) (int, error)
	GetEpisodePlayFID(imdbID string, season, episode int) (int, error)
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
	shareKeyTTL     = 6 * time.Hour
)

type titleCacheEntry struct {
	details meta.TitleDetails
	expiry  time.Time
}

type shareKeyCacheEntry struct {
	key    string
	expiry time.Time
}

// Client is the entry point for catalogue search, VOD browsing, and live TV.
type Client struct {
	showbox *showbox.Client
	febbox  febboxBrowser

	liveCatalog *catalog.Client
	liveGuide   *guide.Client
	liveSports  *livesports.Client
	liveSource  *source.Resolver

	imdb  *imdb.Client
	tmdb  *tmdb.Client
	intro introFetcher

	titleMu     sync.Mutex
	titleGroup  singleflight.Group
	showTitles  map[int]titleCacheEntry
	movieTitles map[int]titleCacheEntry

	shareKeyMu    sync.Mutex
	shareKeyGroup singleflight.Group
	shareKeys     map[string]shareKeyCacheEntry
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

	// FMHY-evaluated live TV sources: DaddyLive, NTV, Pluto, Vavoo.
	liveSource := source.Default()
	liveSports := livesports.New(liveCatalog, liveSource)

	return &Client{

		showbox: showbox.New(showbox.Options{ChildMode: cfg.childMode}),
		febbox:  febbox.NewCached(febboxClient),

		liveCatalog: liveCatalog,
		liveGuide:   liveGuide,
		liveSports:  liveSports,
		liveSource:  liveSource,

		imdb:  imdb.New(cfg.tmdbAPIKey),
		tmdb:  tmdb.New(cfg.tmdbAPIKey),
		intro: intro,

		showTitles:  make(map[int]titleCacheEntry),
		movieTitles: make(map[int]titleCacheEntry),
		shareKeys:   make(map[string]shareKeyCacheEntry),
	}

}

// TMDB returns the TMDB discover client.
func (c *Client) TMDB() *tmdb.Client {

	return c.tmdb

}

// SearchKind queries Showbox limited to movies or TV series.
func (c *Client) SearchKind(query string, kind meta.MediaKind) ([]meta.SearchHit, error) {

	mediaType := showbox.MediaMovie

	if kind == meta.MediaShow {

		mediaType = showbox.MediaTV

	}

	results, err := c.showbox.Search(query, mediaType, 1, 20)

	hits := make([]meta.SearchHit, 0, 20)

	appendHits := func(results []showbox.SearchResult, assumeKind meta.MediaKind) {

		for _, result := range results {

			hit := meta.HitFromResult(result)

			// Typed Search5 responses often omit box_type (0). Treat as the
			// requested kind rather than dropping every candidate.
			if hit.Kind == 0 {

				hit.Kind = assumeKind

			}

			if hit.Kind != kind {

				continue

			}

			hits = append(hits, hit)

		}

	}

	if err == nil {

		appendHits(results, kind)

	}

	// Typed TV search is unreliable on Showbox — always merge an untyped pass.
	if kind == meta.MediaShow || len(hits) == 0 {

		all, allErr := c.showbox.Search(query, showbox.MediaAll, 1, 20)

		if allErr == nil {

			appendHits(all, kind)

		}

	}

	seen := make(map[int]struct{}, len(hits))
	out := make([]meta.SearchHit, 0, len(hits))

	for _, hit := range hits {

		if _, ok := seen[hit.ID]; ok {

			continue

		}

		seen[hit.ID] = struct{}{}
		out = append(out, hit)

	}

	if len(out) == 0 && err != nil {

		return nil, err

	}

	return out, nil

}

// Warmup starts background live TV catalog refresh.
func (c *Client) Warmup() {

	c.liveCatalog.Warmup()

}

// Search queries the Showbox catalogue for movies and TV shows.
func (c *Client) Search(query string) ([]meta.SearchHit, error) {

	results, err := c.showbox.Search(query, showbox.MediaAll, 1, 25)

	if err != nil {

		return nil, err

	}

	hits := make([]meta.SearchHit, len(results))

	for i, result := range results {

		hits[i] = meta.HitFromResult(result)

	}

	return hits, nil

}

// Movie returns a chainable handle for a movie by Showbox id.
func (c *Client) Movie(id int) *vod.Movie {

	return vod.NewMovie(c, id)

}

// Show returns a chainable handle for a TV series by Showbox id.
func (c *Client) Show(id int) *vod.Show {

	return vod.NewShow(c, id)

}

// MovieFromHit returns a Movie handle from a search result.
func (c *Client) MovieFromHit(hit meta.SearchHit) (*vod.Movie, error) {

	if hit.Kind != meta.MediaMovie {

		return nil, fmt.Errorf("hit %q is not a movie", hit.Title)

	}

	return c.Movie(hit.ID), nil

}

// ShowFromHit returns a Show handle from a search result.
func (c *Client) ShowFromHit(hit meta.SearchHit) (*vod.Show, error) {

	if hit.Kind == meta.MediaMovie {

		return nil, fmt.Errorf("hit %q is a movie, not a show", hit.Title)

	}

	return c.Show(hit.ID), nil

}

// Channels returns the current live TV channel catalog (metadata only).
func (c *Client) Channels() (*catalog.Catalog, error) {

	return c.liveCatalog.List()

}

// SearchChannels finds channels whose name contains query.
func (c *Client) SearchChannels(query string, limit int) ([]catalog.Channel, error) {

	cat, err := c.liveCatalog.List()

	if err != nil {

		return nil, err

	}

	return cat.Search(query, limit), nil

}

// ResolveChannel resolves a playable stream for a catalog channel id via the
// source-provider layer. With no providers registered, returns source.ErrNoProviders.
func (c *Client) ResolveChannel(channelID string) (string, error) {

	stream, err := c.ResolveChannelStream(channelID)

	if err != nil {

		return "", err

	}

	return stream.URL, nil

}

// ResolveChannelStream resolves a full stream reference (URL + headers) for a channel.
// providerKey is an optional public anonymized key ("auto", "s1", …); empty means auto.
func (c *Client) ResolveChannelStream(channelID string, providerKey ...string) (source.Stream, error) {

	if name, ok := livesports.DecodeChannelID(channelID); ok {
		// Synthetic sports identities are created only after confirming an NTV
		// inventory match. Keep them pinned to that provider so another fuzzy
		// provider cannot resolve the team name to an unrelated channel.
		return c.liveSource.ResolveWith(context.Background(), source.Request{
			ChannelID: channelID,
			Name:      name,
		}, "s2")
	}

	cat, err := c.liveCatalog.List()

	if err != nil {

		return source.Stream{}, err

	}

	ch, ok := cat.FindByID(channelID)

	if !ok {

		return source.Stream{}, fmt.Errorf("live: channel %q not found", channelID)

	}

	key := ""

	if len(providerKey) > 0 {

		key = providerKey[0]

	}

	return c.liveSource.ResolveWith(context.Background(), source.Request{

		ChannelID: ch.ID,
		Name:      ch.Name,
		AltNames:  append([]string(nil), ch.AltNames...),
		Network:   ch.Network,
		Country:   ch.Country.Code,
	}, key)

}

// LiveSourceProviders returns anonymized live source options for the client UI.
func (c *Client) LiveSourceProviders() []source.PublicProvider {

	return c.liveSource.PublicProviders()

}

// LiveSchedule returns guide entries for popular channels.
func (c *Client) LiveSchedule() ([]guide.Entry, error) {

	return c.liveGuide.Schedule()

}

// Matches returns current and upcoming sports fixtures from scoreboard sources.
func (c *Client) Matches() ([]livesports.Match, error) {

	return c.liveSports.Matches()

}

// Trending returns hot search keywords from Showbox.
func (c *Client) Trending(kind meta.MediaKind, limit int) ([]string, error) {

	return discover.Trending(c, kind, limit)

}

// TopCategories returns curated ranking categories for movies or TV.
func (c *Client) TopCategories(kind meta.MediaKind) ([]discover.TopCategory, error) {

	return discover.TopCategories(c, kind)

}

// FebboxDownloadURL resolves a direct download link for a file in a Febbox share.
func (c *Client) FebboxDownloadURL(shareKey string, fid int) (string, error) {

	return c.febbox.GetDownloadURL(shareKey, fid, "")

}

// --- vod.Deps implementation ---

func (c *Client) cachedTitleDetails(items map[int]titleCacheEntry, id int) (meta.TitleDetails, bool) {

	c.titleMu.Lock()
	defer c.titleMu.Unlock()

	entry, ok := items[id]

	if !ok || time.Now().After(entry.expiry) {

		return meta.TitleDetails{}, false

	}

	return entry.details, true

}

func (c *Client) storeTitleDetails(items map[int]titleCacheEntry, id int, details meta.TitleDetails) {

	c.titleMu.Lock()
	defer c.titleMu.Unlock()

	items[id] = titleCacheEntry{details: details, expiry: time.Now().Add(titleDetailsTTL)}

}

func (c *Client) GetMovieDetails(id int) (meta.TitleDetails, error) {

	if details, ok := c.cachedTitleDetails(c.movieTitles, id); ok {

		return details, nil

	}

	result, err, _ := c.titleGroup.Do(fmt.Sprintf("movie:%d", id), func() (any, error) {

		if details, ok := c.cachedTitleDetails(c.movieTitles, id); ok {

			return details, nil

		}

		raw, err := c.showbox.GetMovie(id)

		if err != nil {

			return meta.TitleDetails{}, err

		}

		details := meta.ParseTitleDetails(raw)

		if details.IMDBId != "" {

			if m, err := c.imdb.Movie(details.IMDBId); err == nil {

				meta.EnrichTitleDetails(&details, m)

			}

		}

		c.storeTitleDetails(c.movieTitles, id, details)

		return details, nil

	})

	if err != nil {

		return meta.TitleDetails{}, err

	}

	return result.(meta.TitleDetails), nil

}

func (c *Client) GetShowDetails(id int) (meta.TitleDetails, error) {

	if details, ok := c.cachedTitleDetails(c.showTitles, id); ok {

		return details, nil

	}

	result, err, _ := c.titleGroup.Do(fmt.Sprintf("show:%d", id), func() (any, error) {

		if details, ok := c.cachedTitleDetails(c.showTitles, id); ok {

			return details, nil

		}

		raw, err := c.showbox.GetShow(id)

		if err != nil {

			return meta.TitleDetails{}, err

		}

		details := meta.ParseTitleDetails(raw)

		if details.IMDBId != "" {

			if m, err := c.imdb.Series(details.IMDBId); err == nil {

				meta.EnrichTitleDetails(&details, m)

			}

		}

		c.storeTitleDetails(c.showTitles, id, details)

		return details, nil

	})

	if err != nil {

		return meta.TitleDetails{}, err

	}

	return result.(meta.TitleDetails), nil

}

func (c *Client) GetEpisodeMeta(imdbID string, season, episode int) (vod.EpisodeInfo, bool) {

	m, ok := c.imdb.Episode(imdbID, season, episode)

	if !ok {

		return vod.EpisodeInfo{}, false

	}

	return vod.EpisodeInfo{

		Title:       m.Title,
		Description: m.Description,

		Poster: m.Poster,
	}, true

}

func (c *Client) GetSeasonEpisodes(imdbID string, season int) map[int]vod.EpisodeInfo {

	episodes := c.imdb.SeasonEpisodes(imdbID, season)
	out := make(map[int]vod.EpisodeInfo, len(episodes))

	for number, m := range episodes {

		out[number] = vod.EpisodeInfo{

			Title:       m.Title,
			Description: m.Description,

			Poster: m.Poster,
		}

	}

	return out

}

func (c *Client) GetFebBoxID(id, boxType int) (string, error) {

	key := fmt.Sprintf("%d:%d", boxType, id)

	c.shareKeyMu.Lock()

	if entry, ok := c.shareKeys[key]; ok && time.Now().Before(entry.expiry) {

		shareKey := entry.key
		c.shareKeyMu.Unlock()
		return shareKey, nil

	}

	c.shareKeyMu.Unlock()

	result, err, _ := c.shareKeyGroup.Do(key, func() (any, error) {

		c.shareKeyMu.Lock()

		if entry, ok := c.shareKeys[key]; ok && time.Now().Before(entry.expiry) {

			shareKey := entry.key
			c.shareKeyMu.Unlock()
			return shareKey, nil

		}

		c.shareKeyMu.Unlock()

		shareKey, err := c.showbox.GetFebBoxID(id, showbox.BoxType(boxType))

		if err != nil {

			return "", err

		}

		if shareKey != "" {

			c.shareKeyMu.Lock()
			c.shareKeys[key] = shareKeyCacheEntry{key: shareKey, expiry: time.Now().Add(shareKeyTTL)}
			c.shareKeyMu.Unlock()

		}

		return shareKey, nil

	})

	if err != nil {

		return "", err

	}

	return result.(string), nil

}

func (c *Client) GetConsoleMovieFID(imdbID string) (int, error) {

	return c.febbox.GetMoviePlayFID(imdbID)

}

func (c *Client) GetConsoleEpisodeFID(imdbID string, season, episode int) (int, error) {

	return c.febbox.GetEpisodePlayFID(imdbID, season, episode)

}

func (c *Client) GetConsoleLinks(fid int) ([]febbox.Quality, error) {

	return c.febbox.GetConsoleLinks(fid)

}

func (c *Client) ListFiles(shareKey string, parentID any, cookie string) ([]febbox.File, error) {

	return c.febbox.ListFiles(shareKey, parentID, cookie)

}

func (c *Client) GetLinks(shareKey string, fid any, cookie string) ([]febbox.Quality, error) {

	return c.febbox.GetLinks(shareKey, fid, cookie)

}

func (c *Client) GetDownloadURL(shareKey string, fid any, cookie string) (string, error) {

	return c.febbox.GetDownloadURL(shareKey, fid, cookie)

}

func (c *Client) GetIntro(query introdb.MediaQuery) (*introdb.MediaRecord, error) {

	return c.intro.GetMedia(query)

}

func (c *Client) GetShowSeasonsByTMDB(tmdbID int) ([]vod.ShowSeasonInfo, error) {

	summaries, err := c.imdb.ShowSeasonsByTMDBID(tmdbID)

	if err != nil {

		return nil, err

	}

	out := make([]vod.ShowSeasonInfo, len(summaries))

	for i, s := range summaries {

		out[i] = vod.ShowSeasonInfo{

			Number:       s.Number,
			EpisodeCount: s.EpisodeCount,
			Name:         s.Name,
		}

	}

	return out, nil

}

// --- discover.Deps implementation ---

func (c *Client) TopHot(mediaType showbox.MediaType, limit int) ([]string, error) {

	return c.showbox.TopHot(mediaType, limit)

}

func (c *Client) TopLists(boxType showbox.BoxType) ([]showbox.TopList, error) {

	return c.showbox.TopLists(boxType)

}

func (c *Client) TopListMovies(listID string, page, limit int) ([]showbox.SearchResult, error) {

	return c.showbox.TopListMovies(listID, page, limit)

}

func (c *Client) TopListTV(listID string, page, limit int) ([]showbox.SearchResult, error) {

	return c.showbox.TopListTV(listID, page, limit)

}
