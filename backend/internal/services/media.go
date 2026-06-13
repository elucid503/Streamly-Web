package services

import (
	"context"
	"sync"
	"time"

	mediakit "mediakit"
	"streamly/internal/config"
)

type MediaService struct {
	client *mediakit.Client
	cfg    *config.Config

	catalogMu   sync.RWMutex
	catalog     catalogSnapshot
	cacheCancel context.CancelFunc

	upstreamMu   sync.Mutex
	lastUpstream time.Time
	batchMu      sync.RWMutex
	batch        *refreshBatch

	searchCacheMu sync.RWMutex
	searchCache   map[string]searchCacheEntry

	vodMu         sync.RWMutex
	seasonsCache  map[int]vodCacheEntry[[]SeasonDTO]
	episodesCache map[string]vodCacheEntry[[]EpisodeDTO]

	qualitiesMu           sync.RWMutex
	movieQualitiesCache   map[int]qualitiesCacheEntry
	episodeQualitiesCache map[string]qualitiesCacheEntry
}

func NewMediaService(cfg *config.Config) *MediaService {
	client := mediakit.New(
		mediakit.WithFebboxCookie(cfg.FebboxCookie),
		mediakit.WithIntroDBKey(cfg.IntroDBKey),
		mediakit.WithTVBaseURL(cfg.TVBaseURL),
		mediakit.WithChildMode(cfg.ChildMode),
		mediakit.WithIntroCache(true),
	)
	client.Warmup()
	return &MediaService{
		client: client,
		cfg:    cfg,
		catalog: catalogSnapshot{
			movieCategoryTitles: make(map[string][]SearchResultDTO),
			showCategoryTitles:  make(map[string][]SearchResultDTO),
		},
		searchCache:           make(map[string]searchCacheEntry),
		seasonsCache:          make(map[int]vodCacheEntry[[]SeasonDTO]),
		episodesCache:         make(map[string]vodCacheEntry[[]EpisodeDTO]),
		movieQualitiesCache:   make(map[int]qualitiesCacheEntry),
		episodeQualitiesCache: make(map[string]qualitiesCacheEntry),
	}
}

type SearchResultDTO struct {
	ID          int    `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Year        int    `json:"year"`
	Poster      string `json:"poster"`
	Description string `json:"description"`
	Rating      string `json:"rating"`
}

type TitleDetailsDTO struct {
	ID          int    `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Year        string `json:"year"`
	Poster      string `json:"poster"`
	Banner      string `json:"banner,omitempty"`
	Description string `json:"description"`
	Rating      string `json:"rating"`
}

type CategoryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type SeasonDTO struct {
	Number int    `json:"number"`
	Label  string `json:"label"`
}

type EpisodeDTO struct {
	Season      int    `json:"season"`
	Episode     int    `json:"episode"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Poster      string `json:"poster,omitempty"`
}

type QualityDTO struct {
	Label    string `json:"label"`
	Height   int    `json:"height"`
	IsHLS    bool   `json:"isHls"`
	URL      string `json:"url"`
	ProxyURL string `json:"proxyUrl,omitempty"`
}

type StreamDTO struct {
	Qualities      []QualityDTO `json:"qualities"`
	URL            string       `json:"url"`
	ProxyURL       string       `json:"proxyUrl,omitempty"`
	IsHLS          bool         `json:"isHls"`
	SelectedHeight int          `json:"selectedHeight"`
}

type SubtitleDTO struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Language string `json:"language"`
	Format   string `json:"format"`
	ProxyURL string `json:"proxyUrl"`
	Source   string `json:"source,omitempty"`
}

type IntroDTO struct {
	IntroStartMs   *int64 `json:"introStartMs,omitempty"`
	IntroEndMs     *int64 `json:"introEndMs,omitempty"`
	CreditsStartMs *int64 `json:"creditsStartMs,omitempty"`
}

type NextEpisodeDTO struct {
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
	Title   string `json:"title"`
}

type LiveChannelDTO struct {
	ID       string `json:"id"`
	DaddyID  string `json:"daddyId"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Logo     string `json:"logo"`
	Country  string `json:"country"`
	Category string `json:"category"`
}

func (s *MediaService) Client() *mediakit.Client {
	return s.client
}

func (s *MediaService) DefaultHeight() int {
	return s.cfg.DefaultQuality
}

func (s *MediaService) Trending(kind mediakit.MediaKind, limit int) ([]string, error) {
	return s.client.Trending(kind, limit)
}

func (s *MediaService) Categories(kind mediakit.MediaKind) ([]CategoryDTO, error) {
	snap := s.snapshot()
	if kind == mediakit.MediaMovie {
		return append([]CategoryDTO(nil), snap.movieCategories...), nil
	}
	return append([]CategoryDTO(nil), snap.showCategories...), nil
}

func (s *MediaService) CategoryTitles(kind mediakit.MediaKind, categoryID string, page, limit int) ([]SearchResultDTO, error) {
	snap := s.snapshot()
	var titles []SearchResultDTO
	if kind == mediakit.MediaMovie {
		titles = snap.movieCategoryTitles[categoryID]
	} else {
		titles = snap.showCategoryTitles[categoryID]
	}
	return slicePage(titles, page, limit), nil
}

func (s *MediaService) MovieDetails(id int) (*TitleDetailsDTO, error) {
	details, err := s.client.Movie(id).Details()
	if err != nil {
		return nil, err
	}
	return titleDetailsToDTO(id, "movie", details), nil
}

func (s *MediaService) ShowDetails(id int) (*TitleDetailsDTO, error) {
	details, err := s.client.Show(id).Details()
	if err != nil {
		return nil, err
	}
	return titleDetailsToDTO(id, "show", details), nil
}

func (s *MediaService) ShowSeasons(id int) ([]SeasonDTO, error) {
	return s.cachedShowSeasons(id)
}

func (s *MediaService) SeasonEpisodes(showID, season int) ([]EpisodeDTO, error) {
	return s.cachedSeasonEpisodes(showID, season)
}

func (s *MediaService) EpisodeDetails(showID, season, episode int) (*EpisodeDTO, error) {
	ep := s.client.Show(showID).Episode(season, episode)
	info, err := ep.Info()
	if err != nil {
		return nil, err
	}
	return &EpisodeDTO{
		Season:      season,
		Episode:     episode,
		Title:       info.Title,
		Description: info.Description,
		Poster:      info.Poster,
	}, nil
}

func (s *MediaService) MovieQualities(id, height int) ([]mediakit.Quality, *mediakit.Quality, error) {
	qualities, err := s.cachedMovieQualities(id)
	if err != nil {
		return nil, nil, err
	}
	if height <= 0 {
		height = s.cfg.DefaultQuality
	}
	best := mediakit.PickQuality(qualities, height)
	return qualities, best, nil
}

func (s *MediaService) MovieSubtitles(id int) ([]mediakit.Subtitle, error) {
	return s.client.Movie(id).Subtitles()
}

func (s *MediaService) EpisodeSubtitles(showID, season, episode int) ([]mediakit.Subtitle, error) {
	return s.client.Show(showID).Episode(season, episode).Subtitles()
}

func (s *MediaService) EpisodeQualities(showID, season, episode, height int) ([]mediakit.Quality, *mediakit.Quality, error) {
	qualities, err := s.cachedEpisodeQualities(showID, season, episode)
	if err != nil {
		return nil, nil, err
	}
	if height <= 0 {
		height = s.cfg.DefaultQuality
	}
	best := mediakit.PickQuality(qualities, height)
	return qualities, best, nil
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
	next, err := retryUpstream(2, func() (*mediakit.Episode, error) {
		s.throttleUpstream()
		return s.client.Show(showID).NextEpisode(season, episode)
	})
	if err != nil {
		return nil, err
	}
	if next == nil {
		return nil, nil
	}
	title, _ := next.Title()
	return &NextEpisodeDTO{
		Season:  next.SeasonNumber(),
		Episode: next.Number(),
		Title:   title,
	}, nil
}

func (s *MediaService) LiveChannels() ([]LiveChannelDTO, error) {
	snap := s.snapshot()
	return append([]LiveChannelDTO(nil), snap.liveChannels...), nil
}

func (s *MediaService) LivePopular(limit int) ([]LiveChannelDTO, error) {
	snap := s.snapshot()
	return sliceLimit(snap.livePopular, limit), nil
}

func (s *MediaService) LiveSearch(query string, limit int) ([]LiveChannelDTO, error) {
	snap := s.snapshot()
	return filterLiveChannels(snap.liveChannels, query, limit), nil
}

func (s *MediaService) ResolveLiveStream(daddyID string) (*mediakit.LiveStream, error) {
	return s.client.Channel(daddyID).Resolve()
}

func QualitiesToDTO(items []mediakit.Quality) []QualityDTO {
	out := make([]QualityDTO, 0, len(items))
	for _, q := range items {
		out = append(out, QualityDTO{
			Label:  q.Label,
			Height: q.Height,
			IsHLS:  q.IsHLS,
			URL:    q.URL,
		})
	}
	return out
}

func BuildStreamDTO(qualities []mediakit.Quality, best *mediakit.Quality) *StreamDTO {
	if best == nil || best.URL == "" {
		return nil
	}
	return &StreamDTO{
		Qualities:      QualitiesToDTO(qualities),
		URL:            best.URL,
		IsHLS:          best.IsHLS || mediakit.IsHLSURL(best.URL),
		SelectedHeight: best.Height,
	}
}

func hitToDTO(hit mediakit.SearchHit) SearchResultDTO {
	return SearchResultDTO{
		ID:          hit.ID,
		Kind:        kindName(hit.Kind),
		Title:       hit.Title,
		Year:        hit.Year,
		Poster:      hit.Poster,
		Description: hit.Description,
		Rating:      hit.IMDBRating,
	}
}

func titleDetailsToDTO(id int, kind string, details mediakit.TitleDetails) *TitleDetailsDTO {
	return &TitleDetailsDTO{
		ID:          id,
		Kind:        kind,
		Title:       details.Title,
		Year:        details.Year,
		Poster:      details.Poster,
		Banner:      details.Banner,
		Description: details.Description,
		Rating:      details.IMDBRating,
	}
}

func kindName(kind mediakit.MediaKind) string {
	if kind == mediakit.MediaMovie {
		return "movie"
	}
	return "show"
}

func liveChannelToDTO(info mediakit.LiveChannelInfo) LiveChannelDTO {
	return LiveChannelDTO{
		ID:       info.ID,
		DaddyID:  info.DaddyID,
		Name:     info.Name,
		Slug:     info.Slug,
		Logo:     info.Logo,
		Country:  info.Country,
		Category: info.Category,
	}
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
