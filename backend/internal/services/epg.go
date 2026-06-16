package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	epgCacheTTL       = 30 * time.Minute
	tvmazeScheduleURL = "https://api.tvmaze.com/schedule?country=US"
)

// ProgramDTO is a TV program airing slot.
type ProgramDTO struct {

	Title string `json:"title"`
	StartsAt int64 `json:"startsAt"` // Unix seconds
	Runtime int `json:"runtime"`  // minutes
	Image string `json:"image,omitempty"`

}

// ChannelGuideEntry pairs a live channel with its current and next program.
type ChannelGuideEntry struct {

	Channel LiveChannelDTO `json:"channel"`
	Current *ProgramDTO `json:"current,omitempty"`
	Next *ProgramDTO `json:"next,omitempty"`

}

var epgCache struct {

	mu sync.RWMutex
	entries []ChannelGuideEntry

	fetchedAt time.Time

}

type tvmazeItem struct {

	AirStamp string `json:"airstamp"`
	Runtime int `json:"runtime"`

	Show struct {

		Name string `json:"name"`

		Network *struct {

			Name string `json:"name"`

		} `json:"network"`

		WebChannel *struct {

			Name string `json:"name"`

		} `json:"webChannel"`

		Image *struct {

			Medium string `json:"medium"`

		} `json:"image"`

	} `json:"show"`

}

// LiveSchedule returns popular channels with their current and upcoming programs.
func (s *MediaService) LiveSchedule() ([]ChannelGuideEntry, error) {

	epgCache.mu.RLock()

	if time.Since(epgCache.fetchedAt) < epgCacheTTL && epgCache.entries != nil {

		entries := epgCache.entries
		epgCache.mu.RUnlock()

		return entries, nil

	}

	epgCache.mu.RUnlock()

	epgCache.mu.Lock()
	defer epgCache.mu.Unlock()

	if time.Since(epgCache.fetchedAt) < epgCacheTTL && epgCache.entries != nil {

		return epgCache.entries, nil

	}

	items, err := fetchTVMaze()

	if err != nil {

		if epgCache.entries != nil {

			return epgCache.entries, nil

		}

		return nil, err

	}

	channels, err := s.LivePopular(20)

	if err != nil {

		return nil, err

	}

	entries := buildChannelGuide(items, channels)

	epgCache.entries = entries
	epgCache.fetchedAt = time.Now()

	return entries, nil

}

func fetchTVMaze() ([]tvmazeItem, error) {

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(tvmazeScheduleURL)

	if err != nil {

		return nil, fmt.Errorf("epg: fetch schedule: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("epg: schedule status %d", resp.StatusCode)

	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if err != nil {

		return nil, fmt.Errorf("epg: read schedule: %w", err)

	}

	var items []tvmazeItem

	if err := json.Unmarshal(body, &items); err != nil {

		return nil, fmt.Errorf("epg: decode schedule: %w", err)

	}

	return items, nil

}

func buildChannelGuide(items []tvmazeItem, channels []LiveChannelDTO) []ChannelGuideEntry {

	now := time.Now()

	byNetwork := make(map[string][]tvmazeItem)

	for _, item := range items {

		key := tvmazeNetworkKey(item)

		if key == "" {

			continue

		}

		byNetwork[key] = append(byNetwork[key], item)

	}

	var entries []ChannelGuideEntry

	for _, ch := range channels {

		key := matchNetworkKey(ch.Name, byNetwork)

		if key == "" {

			continue

		}

		current, next := findPrograms(byNetwork[key], now)

		if current == nil && next == nil {

			continue

		}

		entries = append(entries, ChannelGuideEntry{

			Channel: ch,

			Current: current,
			Next: next,

		})

	}

	return entries

}

func tvmazeNetworkKey(item tvmazeItem) string {

	if item.Show.Network != nil && item.Show.Network.Name != "" {

		return strings.ToLower(strings.TrimSpace(item.Show.Network.Name))

	}

	if item.Show.WebChannel != nil && item.Show.WebChannel.Name != "" {

		return strings.ToLower(strings.TrimSpace(item.Show.WebChannel.Name))

	}

	return ""

}

func matchNetworkKey(channelName string, networks map[string][]tvmazeItem) string {

	norm := normalizeChannelName(channelName)

	if _, ok := networks[norm]; ok {

		return norm

	}

	for key := range networks {

		if strings.HasPrefix(norm, key) && len(key) >= 3 {

			return key

		}

		if strings.HasPrefix(key, norm) && len(norm) >= 3 {

			return key

		}

	}

	return ""

}

func normalizeChannelName(name string) string {

	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimSuffix(name, " usa")

	if i := strings.Index(name, " ("); i >= 0 {

		name = strings.TrimSpace(name[:i])

	}

	return name

}

func findPrograms(items []tvmazeItem, now time.Time) (*ProgramDTO, *ProgramDTO) {

	var current *ProgramDTO
	var next *ProgramDTO
	var nextStart time.Time

	for _, item := range items {

		if item.Runtime <= 0 {

			continue

		}

		start, err := time.Parse(time.RFC3339, item.AirStamp)
		if err != nil {

			continue

		}

		end := start.Add(time.Duration(item.Runtime) * time.Minute)

		if now.After(start) && now.Before(end) {

			current = &ProgramDTO{

				Title:    item.Show.Name,

				StartsAt: start.Unix(),
				Runtime:  item.Runtime,

				Image:    tvmazeImage(item),

			}

		} else if start.After(now) && (next == nil || start.Before(nextStart)) {

			nextStart = start

			next = &ProgramDTO{

				Title:    item.Show.Name,

				StartsAt: start.Unix(),
				Runtime:  item.Runtime,

				Image:    tvmazeImage(item),

			}

		}

	}

	return current, next

}

func tvmazeImage(item tvmazeItem) string {

	if item.Show.Image != nil {

		return item.Show.Image.Medium

	}

	return ""

}
