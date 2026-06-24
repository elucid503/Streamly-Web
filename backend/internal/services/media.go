package services

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	mediakit "mediakit"
	"streamly/internal/config"
	"streamly/internal/services/catalog"
	"streamly/internal/services/search"
	"streamly/internal/services/stream"
	"streamly/internal/services/vod"

	"golang.org/x/sync/singleflight"
)

const (

	titleDetailsTTL = 6 * time.Hour
	titleDetailsMaxEntries = 1024

)

// Type aliases so callers (handlers, subtitles, proxy) need not import sub-packages.

type SearchResultDTO = catalog.SearchResultDTO
type CategoryDTO = catalog.CategoryDTO
type LiveChannelDTO = catalog.LiveChannelDTO

type SeasonDTO = vod.SeasonDTO
type EpisodeDTO = vod.EpisodeDTO

// MediaService is the central media orchestrator.
type MediaService struct {

	client *mediakit.Client
	cfg *config.Config

	catalog  *catalog.Cache

	search *search.Cache
	stream *stream.Cache

	vod *vod.Cache

	detailsMu sync.RWMutex
	detailsGroup singleflight.Group

	movieDetails map[int]titleDetailsCacheEntry
	showDetails map[int]titleDetailsCacheEntry

}

// TitleDetailsDTO is user-facing metadata for a movie or show.
type TitleDetailsDTO struct {

	ID int `json:"id"`
	Kind string `json:"kind"`

	Title string `json:"title"`
	Year string `json:"year"`

	Poster string `json:"poster"`
	Banner string `json:"banner,omitempty"`

	Description string `json:"description"`
	Rating string `json:"rating"`

}

type titleDetailsCacheEntry struct {

	details *TitleDetailsDTO
	fetchedAt time.Time

}

// QualityDTO is one downloadable rendition of a video file.
type QualityDTO struct {

	Label string `json:"label"`
	Height int `json:"height"`

	IsHLS bool `json:"isHls"`

	URL string `json:"url"`
	ProxyURL string `json:"proxyUrl,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

}

// StreamDTO is the resolved streaming payload returned to clients.
type StreamDTO struct {

	Qualities []QualityDTO `json:"qualities"`

}

// SubtitleDTO describes an external subtitle track.
type SubtitleDTO struct {

	ID string `json:"id"`
	Label string `json:"label"`

	Language string `json:"language"`
	Format string `json:"format"`

	ProxyURL string `json:"proxyUrl"`
	Source string `json:"source,omitempty"`

}

// IntroDTO carries skip-intro timing offsets.
type IntroDTO struct {

	IntroStartMs *int64 `json:"introStartMs,omitempty"`
	IntroEndMs *int64 `json:"introEndMs,omitempty"`

	CreditsStartMs *int64 `json:"creditsStartMs,omitempty"`

}

// NextEpisodeDTO identifies the episode after the current one.
type NextEpisodeDTO struct {

	Season  int `json:"season"`
	Episode int `json:"episode"`

	Title string `json:"title"`

}

func NewMediaService(cfg *config.Config) *MediaService {

	client := mediakit.New(

		mediakit.WithFebboxCookie(cfg.FebboxCookie),
		mediakit.WithIntroDBKey(cfg.IntroDBKey),
		mediakit.WithTMDBAPIKey(cfg.TMDBAPIKey),
		mediakit.WithTVBaseURL(cfg.TVBaseURL),
		mediakit.WithChildMode(cfg.ChildMode),
		mediakit.WithIntroCache(true),
	)

	client.Warmup()

	cat := catalog.New(client, cfg.CatalogCacheTTL, cfg.CatalogCacheFile)

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

func (s *MediaService) MovieDetails(id int) (*TitleDetailsDTO, error) {

	if details, ok := s.cachedTitleDetails(s.movieDetails, id); ok {

		return details, nil

	}

	result, err, _ := s.detailsGroup.Do(fmt.Sprintf("movie:%d", id), func() (any, error) {

		details, err := s.client.Movie(id).Details()

		if err != nil {

			return nil, err

		}

		dto := titleDetailsToDTO(id, "movie", details)

		s.setTitleDetails(true, id, dto)

		return dto, nil

	})

	if err != nil {

		return nil, err

	}

	return cloneTitleDetails(result.(*TitleDetailsDTO)), nil

}

func (s *MediaService) ShowDetails(id int) (*TitleDetailsDTO, error) {

	if details, ok := s.cachedTitleDetails(s.showDetails, id); ok {

		return details, nil

	}

	result, err, _ := s.detailsGroup.Do(fmt.Sprintf("show:%d", id), func() (any, error) {

		details, err := s.client.Show(id).Details()

		if err != nil {

			return nil, err

		}

		dto := titleDetailsToDTO(id, "show", details)

		s.setTitleDetails(false, id, dto)

		return dto, nil

	})

	if err != nil {

		return nil, err

	}

	return cloneTitleDetails(result.(*TitleDetailsDTO)), nil

}

func (s *MediaService) ShowSeasons(id int) ([]SeasonDTO, error) {

	return s.vod.ShowSeasons(id)

}

func (s *MediaService) SeasonEpisodes(showID, season int) ([]EpisodeDTO, error) {

	return s.vod.SeasonEpisodes(showID, season)

}

func (s *MediaService) EpisodeDetails(showID, season, episode int) (*EpisodeDTO, error) {

	ep := s.client.Show(showID).Episode(season, episode)

	info, err := ep.Info()

	if err != nil {

		return nil, err

	}

	return &EpisodeDTO{

		Season:  season,
		Episode: episode,

		Title:       info.Title,
		Description: info.Description,
		Poster:      info.Poster,
	}, nil

}

func (s *MediaService) MovieQualities(id int) ([]mediakit.Quality, error) {

	return s.stream.MovieQualities(id)

}

func (s *MediaService) EpisodeQualities(showID, season, episode int) ([]mediakit.Quality, error) {

	return s.stream.EpisodeQualities(showID, season, episode)

}

func (s *MediaService) MovieIntro(id int, durationMs int64) (*IntroDTO, error) {

	movie := s.client.Movie(id)

	var opts []mediakit.IntroOption

	if durationMs > 0 {

		opts = append(opts, mediakit.WithDuration(time.Duration(durationMs)*time.Millisecond))

	}

	data, err := movie.Intro(opts...)

	if err != nil {

		return nil, err

	}

	return introToDTO(data, func(d time.Duration) (time.Duration, bool) {

		return movie.CreditsStart(d)

	}), nil

}

func (s *MediaService) EpisodeIntro(showID, season, episode int, durationMs int64) (*IntroDTO, error) {

	ep := s.client.Show(showID).Episode(season, episode)

	var opts []mediakit.IntroOption

	if durationMs > 0 {

		opts = append(opts, mediakit.WithDuration(time.Duration(durationMs)*time.Millisecond))

	}

	data, err := ep.Intro(opts...)

	if err != nil {

		return nil, err

	}

	return introToDTO(data, func(d time.Duration) (time.Duration, bool) {

		return ep.CreditsStart(d)

	}), nil

}

func (s *MediaService) NextEpisode(showID, season, episode int) (*NextEpisodeDTO, error) {

	details, err := s.client.GetShowDetails(showID)

	if err != nil {

		return nil, err

	}

	if details.TMDBId <= 0 {

		return nil, nil

	}

	tmdbSeasons, err := s.client.GetShowSeasonsByTMDB(details.TMDBId)

	if err != nil {

		return nil, err

	}

	var currentEpisodeCount, nextSeasonNum int

	for _, sn := range tmdbSeasons {

		if sn.Number == season {

			currentEpisodeCount = sn.EpisodeCount

		}

		if sn.Number > season && (nextSeasonNum == 0 || sn.Number < nextSeasonNum) {

			nextSeasonNum = sn.Number

		}

	}

	var nextSzn, nextEp int

	if episode < currentEpisodeCount {

		nextSzn, nextEp = season, episode+1

	} else if nextSeasonNum > 0 {

		nextSzn, nextEp = nextSeasonNum, 1

	} else {

		return nil, nil

	}

	var title string

	if details.IMDBId != "" {

		if info, ok := s.client.GetEpisodeMeta(details.IMDBId, nextSzn, nextEp); ok {

			title = info.Title

		}

	}

	return &NextEpisodeDTO{

		Season:  nextSzn,
		Episode: nextEp,
		Title:   title,

	}, nil

}

func (s *MediaService) LiveChannels() ([]LiveChannelDTO, error) {

	return s.catalog.LiveChannels(), nil

}

func (s *MediaService) LivePopular(limit int) ([]LiveChannelDTO, error) {

	return s.catalog.LivePopular(limit), nil

}

func (s *MediaService) LiveSearch(query string, limit int) ([]LiveChannelDTO, error) {

	return s.catalog.LiveSearch(query, limit), nil

}

func (s *MediaService) LiveChannel(daddyID string) (LiveChannelDTO, bool) {

	return s.catalog.LiveChannel(daddyID)

}

func (s *MediaService) ResolveLiveStream(daddyID string) (*mediakit.LiveStream, error) {

	return s.client.Channel(daddyID).Resolve()

}

// QualitiesToDTO converts a slice of qualities to DTOs, filtering to web-playable URLs and sorting by height descending.
func QualitiesToDTO(items []mediakit.Quality) []QualityDTO {

	out := make([]QualityDTO, 0, len(items))

	for _, q := range items {

		if q.URL == "" || !mediakit.IsWebPlayableURL(q.URL) {

			continue

		}

		dto := QualityDTO{

			Label:   q.Label,
			Height:  q.Height,
			IsHLS:   q.IsHLS,
			URL:     q.URL,
			Headers: cloneQualityHeaders(q.Headers),

		}

		out = append(out, dto)

	}

	sort.Slice(out, func(i, j int) bool {

		return out[i].Height > out[j].Height

	})

	return out

}

// BuildStreamDTO assembles a StreamDTO from all available qualities.
func BuildStreamDTO(qualities []mediakit.Quality) *StreamDTO {

	dtos := QualitiesToDTO(qualities)

	if len(dtos) == 0 {

		return nil

	}

	return &StreamDTO{

		Qualities: dtos,
	}

}

func titleDetailsToDTO(id int, kind string, details mediakit.TitleDetails) *TitleDetailsDTO {

	return &TitleDetailsDTO{

		ID:   id,
		Kind: kind,

		Title: details.Title,
		Year:  details.Year,

		Poster: details.Poster,
		Banner: details.Banner,

		Description: details.Description,
		Rating:      details.IMDBRating,
	}

}

func (s *MediaService) cachedTitleDetails(items map[int]titleDetailsCacheEntry, id int) (*TitleDetailsDTO, bool) {

	s.detailsMu.RLock()

	entry, ok := items[id]

	s.detailsMu.RUnlock()

	if !ok || time.Since(entry.fetchedAt) >= titleDetailsTTL {

		return nil, false

	}

	return cloneTitleDetails(entry.details), true

}

func (s *MediaService) setTitleDetails(movie bool, id int, details *TitleDetailsDTO) {

	s.detailsMu.Lock()

	defer s.detailsMu.Unlock()

	s.pruneTitleDetailsLocked()

	entry := titleDetailsCacheEntry{

		details:   cloneTitleDetails(details),
		fetchedAt: time.Now(),
	}

	if movie {

		s.movieDetails[id] = entry

	} else {

		s.showDetails[id] = entry

	}

}

func (s *MediaService) pruneTitleDetailsLocked() {

	now := time.Now()

	for id, entry := range s.movieDetails {

		if now.Sub(entry.fetchedAt) >= titleDetailsTTL || len(s.movieDetails) > titleDetailsMaxEntries {

			delete(s.movieDetails, id)

		}

	}

	for id, entry := range s.showDetails {

		if now.Sub(entry.fetchedAt) >= titleDetailsTTL || len(s.showDetails) > titleDetailsMaxEntries {

			delete(s.showDetails, id)

		}

	}

}

func cloneQualityHeaders(headers map[string]string) map[string]string {

	if len(headers) == 0 {

		return nil

	}

	out := make(map[string]string, len(headers))

	for key, value := range headers {

		if value == "" {

			continue

		}

		out[key] = value

	}

	return out

}

func cloneTitleDetails(details *TitleDetailsDTO) *TitleDetailsDTO {

	if details == nil {

		return nil

	}

	cp := *details

	return &cp

}

func introToDTO(data *mediakit.IntroData, creditsFn func(time.Duration) (time.Duration, bool)) *IntroDTO {

	if data == nil {

		return &IntroDTO{}

	}

	dto := &IntroDTO{}

	if start, end, ok := data.IntroWindow(); ok {

		startMs := start.Milliseconds()
		endMs := end.Milliseconds()

		dto.IntroStartMs = &startMs
		dto.IntroEndMs = &endMs

	}

	if creditsFn != nil {

		if credits, ok := creditsFn(0); ok {

			ms := credits.Milliseconds()

			dto.CreditsStartMs = &ms

		}

	}

	return dto

}
