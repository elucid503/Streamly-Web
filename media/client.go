package mediakit

import (
	"fmt"

	"mediakit/internal/febbox"
	"mediakit/internal/introdb"
	"mediakit/internal/showbox"
	"mediakit/internal/imdb"
	"mediakit/internal/tv"
)

type febboxBrowser interface {
	ListFiles(shareKey string, parentID any, cookie string) ([]febbox.File, error)
	GetLinks(shareKey string, fid any, cookie string) ([]febbox.Quality, error)
	GetDownloadURL(shareKey string, fid any, cookie string) (string, error)
}

// Client is the entry point for catalogue search, VOD browsing, and live TV.
type Client struct {
	showbox *showbox.Client
	febbox  febboxBrowser
	tv      *tv.Client
	imdb    *imdb.Client
	intro   introFetcher
}

type introFetcher interface {
	GetMedia(query introdb.MediaQuery) (*introdb.MediaRecord, error)
}

// New builds a Client with optional configuration.
func New(opts ...Option) *Client {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	applyDefaults(cfg)

	introClient := introdb.New(cfg.introDBOptions())
	var intro introFetcher = introClient
	if cfg.cacheIntro {
		intro = introdb.NewCached(introClient)
	}

	febboxClient := febbox.New(cfg.febboxOptions())
	feb := febboxBrowser(febbox.NewCached(febboxClient))

	return &Client{
		showbox: showbox.New(cfg.showboxOptions()),
		febbox:  feb,
		tv:      tv.New(cfg.tvOptions()),
		imdb:    imdb.New(),
		intro:   intro,
	}
}

// Warmup starts background live TV catalog refresh.
func (c *Client) Warmup() {
	c.tv.Warmup()
}

// Search queries the Showbox catalogue for movies and TV shows.
func (c *Client) Search(query string) ([]SearchHit, error) {
	results, err := c.showbox.Search(query, showbox.MediaAll, 1, 25)
	if err != nil {
		return nil, err
	}

	hits := make([]SearchHit, len(results))
	for i, result := range results {
		hits[i] = hitFromResult(result)
	}

	return hits, nil
}

// Show returns a chainable handle for a TV series by Showbox id.
func (c *Client) Show(id int) *Show {
	return &Show{client: c, id: id}
}

// Movie returns a chainable handle for a movie by Showbox id.
func (c *Client) Movie(id int) *Movie {
	return &Movie{client: c, id: id}
}

// ShowFromHit returns a Show handle from a search result.
func (c *Client) ShowFromHit(hit SearchHit) (*Show, error) {
	if hit.Kind == MediaMovie {
		return nil, fmt.Errorf("hit %q is a movie, not a show", hit.Title)
	}
	return c.Show(hit.ID), nil
}

// MovieFromHit returns a Movie handle from a search result.
func (c *Client) MovieFromHit(hit SearchHit) (*Movie, error) {
	if hit.Kind != MediaMovie {
		return nil, fmt.Errorf("hit %q is not a movie", hit.Title)
	}
	return c.Movie(hit.ID), nil
}

// LiveTV returns the live channel catalog for browsing and search.
func (c *Client) LiveTV() (*LiveCatalog, error) {
	catalog, err := c.tv.ListChannels()
	if err != nil {
		return nil, err
	}
	return &LiveCatalog{client: c, raw: catalog}, nil
}

// Channel returns a chainable handle for a live TV channel by daddyId.
func (c *Client) Channel(daddyID string) *LiveChannel {
	return &LiveChannel{client: c, daddyID: daddyID}
}

// FebboxDownloadURL resolves a direct download link for a file in a Febbox share.
func (c *Client) FebboxDownloadURL(shareKey string, fid int) (string, error) {
	return c.febbox.GetDownloadURL(shareKey, fid, "")
}