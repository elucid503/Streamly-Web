package catalog

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

type snapshotFile struct {

	MovieTrending []SearchResultDTO `json:"movieTrending"`
	ShowTrending []SearchResultDTO `json:"showTrending"`

	MovieCategories []CategoryDTO `json:"movieCategories"`
	ShowCategories []CategoryDTO `json:"showCategories"`

	MovieCategoryTitles map[string][]SearchResultDTO `json:"movieCategoryTitles"`
	ShowCategoryTitles map[string][]SearchResultDTO `json:"showCategoryTitles"`

	LiveChannels []LiveChannelDTO `json:"liveChannels"`
	LivePopular []LiveChannelDTO `json:"livePopular"`

	SearchIndex []SearchResultDTO `json:"searchIndex"`
	RefreshedAt time.Time `json:"refreshedAt"`

}

func snapshotToFile(snap Snapshot) snapshotFile {

	return snapshotFile{

		MovieTrending: snap.movieTrending,
		ShowTrending: snap.showTrending,

		MovieCategories: snap.movieCategories,
		ShowCategories: snap.showCategories,

		MovieCategoryTitles: snap.movieCategoryTitles,
		ShowCategoryTitles: snap.showCategoryTitles,

		LiveChannels: snap.liveChannels,
		LivePopular: snap.livePopular,

		SearchIndex: snap.searchIndex,
		RefreshedAt: snap.refreshedAt,

	}

}

func snapshotFromFile(file snapshotFile) Snapshot {

	movieTitles := file.MovieCategoryTitles

	if movieTitles == nil {

		movieTitles = make(map[string][]SearchResultDTO)

	}

	showTitles := file.ShowCategoryTitles

	if showTitles == nil {

		showTitles = make(map[string][]SearchResultDTO)

	}

	return Snapshot{

		movieTrending: file.MovieTrending,
		showTrending: file.ShowTrending,

		movieCategories: file.MovieCategories,
		showCategories: file.ShowCategories,

		movieCategoryTitles: movieTitles,
		showCategoryTitles: showTitles,

		liveChannels: file.LiveChannels,
		livePopular: file.LivePopular,

		searchIndex: file.SearchIndex,
		refreshedAt: file.RefreshedAt,

	}

}

func (c *Cache) loadFromDisk() bool {

	path := c.cacheFile

	if path == "" {

		return false

	}

	data, err := os.ReadFile(path)

	if err != nil {

		if !os.IsNotExist(err) {

			log.Printf("[catalog-cache] failed to read %s: %v", path, err)

		}

		return false

	}

	var file snapshotFile

	if err := json.Unmarshal(data, &file); err != nil {

		log.Printf("[catalog-cache] failed to parse %s: %v", path, err)

		return false

	}

	snap := snapshotFromFile(file)

	if snap.refreshedAt.IsZero() {

		snap.refreshedAt = time.Now()

	}

	if len(snap.searchIndex) == 0 {

		snap.searchIndex = buildSearchIndex(snap)

	}

	c.mu.Lock()

	c.snap = snap

	c.mu.Unlock()

	log.Printf("[catalog-cache] loaded disk cache from %s (%d search index entries, refreshed %s)",
		path, len(snap.searchIndex), snap.refreshedAt.Format(time.RFC3339))

	return true

}

func (c *Cache) saveToDisk(snap Snapshot) {

	path := c.cacheFile

	if path == "" {

		return

	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {

		log.Printf("[catalog-cache] failed to create cache dir: %v", err)

		return

	}

	payload, err := json.MarshalIndent(snapshotToFile(snap), "", "  ")

	if err != nil {

		log.Printf("[catalog-cache] failed to encode cache: %v", err)

		return

	}

	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, payload, 0o644); err != nil {

		log.Printf("[catalog-cache] failed to write temp cache: %v", err)

		return

	}

	if err := os.Rename(tmp, path); err != nil {

		log.Printf("[catalog-cache] failed to rename cache file: %v", err)

		_ = os.Remove(tmp)

		return

	}

}
