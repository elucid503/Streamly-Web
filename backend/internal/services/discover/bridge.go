package discover

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mediakit "mediakit"
	"streamly/internal/services/catalog"
)

func (s *Service) resolve(item mediakit.TMDBItem) (FeedItem, bool) {

	kind := kindName(item.Kind)

	if cached, ok := s.bridge.get(kind, item.ID); ok {

		if !cached.Miss && cached.ShowboxID > 0 {

			return feedFromBridge(cached, item), true

		}

		// Stale miss entries — fall through and retry.

	}

	mediaKind := mediakit.MediaMovie

	if item.Kind == mediakit.TMDBTV {

		mediaKind = mediakit.MediaShow

	}

	hit, how := s.findVerifiedHit(item, mediaKind)

	entry := bridgeEntry{

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

		ResolvedAt: time.Now(),

	}

	if hit.ID == 0 {

		return FeedItem{}, false

	}

	entry.ShowboxID = hit.ID

	if hit.Poster != "" {

		entry.Poster = hit.Poster

	}

	if hit.Description != "" && entry.Description == "" {

		entry.Description = hit.Description

	}

	if hit.IMDBRating != "" {

		entry.Rating = hit.IMDBRating

	}

	if hit.Year > 0 && entry.Year == 0 {

		entry.Year = hit.Year

	}

	s.bridge.put(entry)

	log.Printf("discover: bridged %s %q → showbox=%d via %s", kind, item.Title, hit.ID, how)

	return feedFromBridge(entry, item), true

}

func (s *Service) findVerifiedHit(item mediakit.TMDBItem, mediaKind mediakit.MediaKind) (mediakit.SearchHit, string) {

	title := strings.TrimSpace(item.Title)

	if title == "" || item.ID <= 0 {

		return mediakit.SearchHit{}, ""

	}

	imdbID := s.tmdbIMDB(item.Kind, item.ID)

	queries := []string{title}

	if i := strings.IndexAny(title, ":—"); i > 2 {

		queries = append(queries, strings.TrimSpace(title[:i]))

	}

	seenIDs := make(map[int]struct{})

	var best mediakit.SearchHit
	var bestHow string
	var bestScore int

	consider := func(hit mediakit.SearchHit, how string, score int) {

		if score > bestScore {

			best = hit
			bestHow = how
			bestScore = score

		}

	}

	for _, query := range queries {

		hits := s.searchCandidates(query, mediaKind)

		for i, hit := range hits {

			if i >= 12 {

				break

			}

			if _, ok := seenIDs[hit.ID]; ok {

				continue

			}

			seenIDs[hit.ID] = struct{}{}

			ok, how, score := s.scoreHit(hit, item, mediaKind, imdbID, i)

			if !ok {

				continue

			}

			// Hard ID match — take immediately.
			if how == "tmdb-id" || how == "imdb-id" {

				return hit, how

			}

			consider(hit, how, score)

		}

	}

	if best.ID > 0 {

		return best, bestHow

	}

	return mediakit.SearchHit{}, ""

}

func (s *Service) searchCandidates(query string, mediaKind mediakit.MediaKind) []mediakit.SearchHit {

	seen := make(map[int]struct{})
	out := make([]mediakit.SearchHit, 0, 32)

	add := func(hits []mediakit.SearchHit) {

		for _, hit := range hits {

			if hit.ID <= 0 {

				continue

			}

			if hit.Kind == 0 {

				hit.Kind = mediaKind

			}

			if hit.Kind != mediaKind {

				continue

			}

			if _, ok := seen[hit.ID]; ok {

				continue

			}

			seen[hit.ID] = struct{}{}
			out = append(out, hit)

		}

	}

	add(s.localCandidates(query, mediaKind))

	if s.fallback != nil {

		add(dtoToHits(s.fallback.SearchTitles(query), mediaKind))

	}

	if hits, err := s.client.SearchKind(query, mediaKind); err == nil {

		add(hits)

	}

	if hits, err := s.client.Search(query); err == nil {

		add(hits)

	}

	return out

}

func dtoToHits(items []catalog.SearchResultDTO, mediaKind mediakit.MediaKind) []mediakit.SearchHit {

	wantKind := "movie"

	if mediaKind == mediakit.MediaShow {

		wantKind = "show"

	}

	out := make([]mediakit.SearchHit, 0, len(items))

	for _, item := range items {

		if item.Kind != wantKind || item.ID <= 0 {

			continue

		}

		out = append(out, mediakit.SearchHit{

			ID: item.ID,
			Kind: mediaKind,
			Title: item.Title,
			Year: item.Year,
			Poster: item.Poster,
			Description: item.Description,
			IMDBRating: item.Rating,

		})

	}

	return out

}

func (s *Service) localCandidates(query string, mediaKind mediakit.MediaKind) []mediakit.SearchHit {

	if s.fallback == nil {

		return nil

	}

	index := s.fallback.CatalogIndex()

	if len(index) == 0 {

		return nil

	}

	wantKind := "movie"

	if mediaKind == mediakit.MediaShow {

		wantKind = "show"

	}

	want := normalizeTitle(query)

	if want == "" {

		return nil

	}

	exact := make([]mediakit.SearchHit, 0, 4)
	partial := make([]mediakit.SearchHit, 0, 8)

	for _, item := range index {

		if item.Kind != wantKind {

			continue

		}

		got := normalizeTitle(item.Title)

		if got == "" {

			continue

		}

		hit := mediakit.SearchHit{

			ID: item.ID,
			Kind: mediaKind,
			Title: item.Title,
			Year: item.Year,
			Poster: item.Poster,
			Description: item.Description,
			IMDBRating: item.Rating,

		}

		if got == want {

			exact = append(exact, hit)

		} else if strings.Contains(got, want) || strings.Contains(want, got) {

			partial = append(partial, hit)

		}

	}

	if len(exact) > 0 {

		return append(exact, partial...)

	}

	return partial

}

// scoreHit ranks a Showbox candidate. Detail lookups are optional — Showbox
// detail endpoints frequently 503, so title matching must work alone.
func (s *Service) scoreHit(hit mediakit.SearchHit, item mediakit.TMDBItem, kind mediakit.MediaKind, wantIMDB string, rank int) (bool, string, int) {

	exact := titlesExact(item.Title, hit.Title)
	compat := titlesCompatible(item.Title, hit.Title)
	yearOK := yearsClose(item.Year, hit.Year)

	if !compat {

		return false, "", 0

	}

	// Exact title is enough to accept without a detail round-trip.
	if exact {

		score := 100 - rank

		if item.Year > 0 && hit.Year > 0 && yearOK {

			return true, "title-year", score + 20

		}

		return true, "title", score

	}

	// Compatible title + year agreement (e.g. "Narcos: Mexico" vs query).
	if item.Year > 0 && hit.Year > 0 && yearOK {

		return true, "fuzzy-year", 60 - rank

	}

	// Top search hit with compatible title — common for TV.
	if rank == 0 {

		return true, "top-result", 50

	}

	// Try ID verification when details are available (best accuracy).
	var details mediakit.TitleDetails
	var err error

	if kind == mediakit.MediaMovie {

		details, err = s.client.GetMovieDetails(hit.ID)

	} else {

		details, err = s.client.GetShowDetails(hit.ID)

	}

	if err != nil {

		return false, "", 0

	}

	if details.TMDBId > 0 && details.TMDBId == item.ID {

		return true, "tmdb-id", 200

	}

	gotIMDB := normalizeIMDB(details.IMDBId)

	if wantIMDB != "" && gotIMDB != "" && wantIMDB == gotIMDB {

		return true, "imdb-id", 180

	}

	return false, "", 0

}

var (

	imdbMu sync.Mutex
	imdbCache = map[string]string{}

)

func (s *Service) tmdbIMDB(kind mediakit.TMDBKind, tmdbID int) string {

	key := string(kind) + ":" + fmt.Sprint(tmdbID)

	imdbMu.Lock()

	if id, ok := imdbCache[key]; ok {

		imdbMu.Unlock()
		return id

	}

	imdbMu.Unlock()

	tmdb := s.client.TMDB()

	if tmdb == nil || !tmdb.Enabled() {

		return ""

	}

	id, err := tmdb.ExternalIMDB(kind, tmdbID)

	if err != nil {

		return ""

	}

	id = normalizeIMDB(id)

	imdbMu.Lock()
	imdbCache[key] = id
	imdbMu.Unlock()

	return id

}

func normalizeIMDB(id string) string {

	id = strings.TrimSpace(strings.ToLower(id))

	if id == "" || id == "null" || id == "<nil>" {

		return ""

	}

	return id

}

func titlesCompatible(want, got string) bool {

	a := normalizeTitle(want)
	b := normalizeTitle(got)

	if a == "" || b == "" {

		return false

	}

	if a == b {

		return true

	}

	if strings.Contains(a, b) || strings.Contains(b, a) {

		return true

	}

	return false

}

func titlesExact(want, got string) bool {

	a := normalizeTitle(want)
	b := normalizeTitle(got)

	return a != "" && a == b

}

func yearsClose(a, b int) bool {

	if a <= 0 || b <= 0 {

		return true

	}

	diff := a - b

	if diff < 0 {

		diff = -diff

	}

	return diff <= 1

}

func feedFromBridge(entry bridgeEntry, item mediakit.TMDBItem) FeedItem {

	poster := entry.Poster

	if poster == "" {

		poster = item.Poster

	}

	backdrop := entry.Backdrop

	if backdrop == "" {

		backdrop = item.Backdrop

	}

	desc := entry.Description

	if desc == "" {

		desc = item.Overview

	}

	rating := entry.Rating

	if rating == "" {

		rating = formatRating(item.VoteAverage)

	}

	genres := entry.Genres

	if len(genres) == 0 {

		genres = mediakit.TMDBGenreNames(item.GenreIDs)

	}

	year := entry.Year

	if year == 0 {

		year = item.Year

	}

	return FeedItem{

		ID: entry.ShowboxID,
		TMDBID: entry.TMDBID,
		Kind: entry.Kind,

		Title: coalesce(entry.Title, item.Title),
		Year: year,

		Poster: poster,
		Backdrop: backdrop,
		Description: desc,
		Rating: rating,

		Genres: genres,
		Runtime: entry.Runtime,

	}

}

func kindName(kind mediakit.TMDBKind) string {

	if kind == mediakit.TMDBTV {

		return "show"

	}

	return "movie"

}

func formatRating(v float64) string {

	if v <= 0 {

		return ""

	}

	return fmt.Sprintf("%.1f", v)

}

func coalesce(a, b string) string {

	if strings.TrimSpace(a) != "" {

		return a

	}

	return b

}

func mediaKindOf(kind mediakit.TMDBKind) mediakit.MediaKind {

	if kind == mediakit.TMDBTV {

		return mediakit.MediaShow

	}

	return mediakit.MediaMovie

}
