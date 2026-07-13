package discover

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	mediakit "mediakit"
	"streamly/internal/models"
)

// GenreWeight is a weighted genre preference.
type GenreWeight struct {

	ID int
	Name string
	Weight float64

}

// TasteProfile summarizes a user's preferences for one media kind.
type TasteProfile struct {

	HasData bool
	TopGenres []GenreWeight
	SeedTMDBID int
	SeedTitle string
	SeedShowboxID int

}

func (s *Service) computeTaste(kind string, history []models.WatchHistoryItem, favorites []models.FavoriteItem) TasteProfile {

	cacheKey := kind

	if len(history) > 0 {

		cacheKey += ":" + history[0].ID.Hex() + ":" + strconv.Itoa(len(history))

	} else if len(favorites) > 0 {

		cacheKey += ":fav:" + favorites[0].ID.Hex() + ":" + strconv.Itoa(len(favorites))

	}

	s.tasteMu.Lock()

	if entry, ok := s.taste[cacheKey]; ok && time.Now().Before(entry.expiry) {

		profile := entry.profile
		s.tasteMu.Unlock()
		return profile

	}

	s.tasteMu.Unlock()

	profile := s.buildTaste(kind, history, favorites)

	s.tasteMu.Lock()
	s.taste[cacheKey] = tasteCacheEntry{profile: profile, expiry: time.Now().Add(tasteCacheTTL)}
	s.tasteMu.Unlock()

	return profile

}

func (s *Service) buildTaste(kind string, history []models.WatchHistoryItem, favorites []models.FavoriteItem) TasteProfile {

	genreScores := make(map[int]float64)
	var seedTMDBID int
	var seedTitle string
	var seedShowboxID int
	var seedWeight float64

	type weighted struct {

		mediaID int
		title string
		year int
		weight float64

	}

	candidates := make([]weighted, 0, 24)

	for _, item := range history {

		if item.Kind != kind || item.MediaID <= 0 {

			continue

		}

		days := time.Since(item.UpdatedAt).Hours() / 24
		decay := math.Pow(0.95, math.Max(0, days))

		weight := 0.4

		if item.Completed {

			weight = 1.0

		} else if item.DurationMs > 0 && float64(item.PositionMs)/float64(item.DurationMs) > 0.2 {

			weight = 0.7

		}

		candidates = append(candidates, weighted{

			mediaID: item.MediaID,
			title: item.Title,
			weight: weight * decay,

		})

	}

	for _, fav := range favorites {

		if fav.Kind != kind || fav.MediaID <= 0 {

			continue

		}

		candidates = append(candidates, weighted{

			mediaID: fav.MediaID,
			title: fav.Title,
			year: fav.Year,
			weight: 1.5,

		})

	}

	if len(candidates) == 0 {

		return TasteProfile{}

	}

	lookups := candidates

	if len(lookups) > 15 {

		lookups = append([]weighted(nil), candidates...)

		sort.Slice(lookups, func(i, j int) bool {

			return lookups[i].weight > lookups[j].weight

		})

		lookups = lookups[:15]

	}

	for _, c := range lookups {

		tmdbID, genres := s.resolveTasteMeta(kind, c.mediaID, c.title, c.year)

		if tmdbID <= 0 {

			continue

		}

		for _, gid := range genres {

			genreScores[gid] += c.weight

		}

		if c.weight > seedWeight {

			seedWeight = c.weight
			seedTMDBID = tmdbID
			seedTitle = c.title
			seedShowboxID = c.mediaID

		}

	}

	if len(genreScores) == 0 && seedTMDBID == 0 {

		return TasteProfile{}

	}

	type pair struct {

		id int
		w float64

	}

	pairs := make([]pair, 0, len(genreScores))

	for id, w := range genreScores {

		pairs = append(pairs, pair{id: id, w: w})

	}

	sort.Slice(pairs, func(i, j int) bool {

		return pairs[i].w > pairs[j].w

	})

	top := make([]GenreWeight, 0, 3)

	for i, p := range pairs {

		if i >= 3 {

			break

		}

		name := mediakit.TMDBGenreName(p.id)

		if name == "" {

			continue

		}

		top = append(top, GenreWeight{ID: p.id, Name: name, Weight: p.w})

	}

	return TasteProfile{

		HasData: len(top) > 0 || seedTMDBID > 0,
		TopGenres: top,
		SeedTMDBID: seedTMDBID,
		SeedTitle: strings.TrimSpace(seedTitle),
		SeedShowboxID: seedShowboxID,

	}

}

func (s *Service) resolveTasteMeta(kind string, showboxID int, title string, year int) (tmdbID int, genres []int) {

	tmdbID, genres = s.lookupGenres(kind, showboxID)

	if tmdbID > 0 {

		return tmdbID, genres

	}

	// Showbox sometimes omits tmdb_id — resolve via TMDB search by title.
	return s.searchTMDBByTitle(kind, title, year)

}

func (s *Service) lookupGenres(kind string, showboxID int) (tmdbID int, genres []int) {

	var details mediakit.TitleDetails
	var err error

	if kind == "movie" {

		details, err = s.client.GetMovieDetails(showboxID)

	} else {

		details, err = s.client.GetShowDetails(showboxID)

	}

	if err != nil || details.TMDBId <= 0 {

		return 0, nil

	}

	tmdbKind := mediakit.TMDBMovie

	if kind == "show" {

		tmdbKind = mediakit.TMDBTV

	}

	item, err := s.client.TMDB().Details(tmdbKind, details.TMDBId)

	if err != nil {

		return details.TMDBId, nil

	}

	return details.TMDBId, item.GenreIDs

}

func (s *Service) searchTMDBByTitle(kind, title string, year int) (tmdbID int, genres []int) {

	tmdb := s.client.TMDB()

	if tmdb == nil || !tmdb.Enabled() || strings.TrimSpace(title) == "" {

		return 0, nil

	}

	tmdbKind := mediakit.TMDBMovie

	if kind == "show" {

		tmdbKind = mediakit.TMDBTV

	}

	items, err := tmdb.Search(tmdbKind, title, year, 5)

	if err != nil || len(items) == 0 {

		// Retry without year constraint.
		if year > 0 {

			items, err = tmdb.Search(tmdbKind, title, 0, 5)

		}

		if err != nil || len(items) == 0 {

			return 0, nil

		}

	}

	want := normalizeTitle(title)

	for _, item := range items {

		if normalizeTitle(item.Title) == want || strings.Contains(normalizeTitle(item.Title), want) {

			return item.ID, item.GenreIDs

		}

	}

	return items[0].ID, items[0].GenreIDs

}
