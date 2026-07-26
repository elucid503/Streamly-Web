package guide

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"mediakit/internal/live/catalog"
)

const (
	scheduleTTL = 20 * time.Minute
	fetchTimeout = 15 * time.Second

	tvmazeBroadcastURL = "https://api.tvmaze.com/schedule?country=US"
	tvmazeWebURL = "https://api.tvmaze.com/schedule/web?country=US"

	browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
)

// Client builds an electronic program guide from TVMaze schedules
// joined to the metadata catalog by network / channel name.
type Client struct {

	httpClient *http.Client
	catalog *catalog.Client

	mu sync.RWMutex
	entries []Entry
	fetchedAt time.Time

}

// New builds a guide Client. catalog supplies channel identity for joins.
func New(cat *catalog.Client) *Client {

	return &Client{

		httpClient: &http.Client{Timeout: fetchTimeout},
		catalog: cat,

	}

}

// Schedule returns guide entries for channels that have matching program data.
func (c *Client) Schedule() ([]Entry, error) {

	c.mu.RLock()

	if time.Since(c.fetchedAt) < scheduleTTL && c.entries != nil {

		out := append([]Entry(nil), c.entries...)
		c.mu.RUnlock()

		return out, nil

	}

	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(c.fetchedAt) < scheduleTTL && c.entries != nil {

		return append([]Entry(nil), c.entries...), nil

	}

	entries, err := c.build()

	if err != nil {

		if c.entries != nil {

			return append([]Entry(nil), c.entries...), nil

		}

		return nil, err

	}

	c.entries = entries
	c.fetchedAt = time.Now()

	return append([]Entry(nil), entries...), nil

}

func (c *Client) build() ([]Entry, error) {

	items, err := c.fetchSchedules()

	if err != nil {

		return nil, err

	}

	cat, err := c.catalog.List()

	if err != nil {

		return nil, err

	}

	byNetwork := indexByNetwork(items)
	channels := cat.Popular(80)

	if len(channels) == 0 {

		channels = cat.Sorted()

		if len(channels) > 80 {

			channels = channels[:80]

		}

	}

	now := time.Now()
	var entries []Entry

	for _, ch := range channels {

		programs := programsForChannel(ch, byNetwork)

		if len(programs) == 0 {

			continue

		}

		current, next, upcoming := splitPrograms(programs, now)

		if current == nil && next == nil {

			continue

		}

		entries = append(entries, Entry{

			Channel: ch,
			Current: current,
			Next: next,
			Upcoming: upcoming,

		})

	}

	return entries, nil

}

type tvmazeItem struct {

	Name string `json:"name"`
	Season int `json:"season"`
	Number int `json:"number"`
	AirStamp string `json:"airstamp"`
	Runtime int `json:"runtime"`
	Summary string `json:"summary"`

	Show struct {

		Name string `json:"name"`
		Genres []string `json:"genres"`
		Summary string `json:"summary"`

		Network *struct {

			Name string `json:"name"`

		} `json:"network"`

		WebChannel *struct {

			Name string `json:"name"`

		} `json:"webChannel"`

		Image *struct {

			Medium string `json:"medium"`
			Original string `json:"original"`

		} `json:"image"`

		Rating *struct {

			Average *float64 `json:"average"`

		} `json:"rating"`

	} `json:"show"`

}

func (c *Client) fetchSchedules() ([]tvmazeItem, error) {

	today := time.Now().Format("2006-01-02")
	urls := []string{
		tvmazeBroadcastURL,
		tvmazeWebURL + "&date=" + today,
	}

	var (
		mu sync.Mutex
		all []tvmazeItem
		errs []error
		wg sync.WaitGroup
	)

	for _, rawURL := range urls {

		wg.Add(1)

		go func(u string) {

			defer wg.Done()

			items, err := c.fetchURL(u)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {

				errs = append(errs, err)
				return

			}

			all = append(all, items...)

		}(rawURL)

	}

	wg.Wait()

	if len(all) == 0 && len(errs) > 0 {

		return nil, fmt.Errorf("guide: schedule fetch failed: %v", errs[0])

	}

	return all, nil

}

func (c *Client) fetchURL(rawURL string) ([]tvmazeItem, error) {

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)

	if err != nil {

		return nil, err

	}

	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("status %d", resp.StatusCode)

	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	if err != nil {

		return nil, err

	}

	var items []tvmazeItem

	if err := json.Unmarshal(body, &items); err != nil {

		return nil, err

	}

	return items, nil

}

func indexByNetwork(items []tvmazeItem) map[string][]Program {

	byNetwork := make(map[string][]Program)

	for _, item := range items {

		key := networkKey(item)

		if key == "" {

			continue

		}

		prog, ok := programFromItem(item)

		if !ok {

			continue

		}

		byNetwork[key] = append(byNetwork[key], prog)

	}

	return byNetwork

}

func networkKey(item tvmazeItem) string {

	if item.Show.Network != nil && item.Show.Network.Name != "" {

		return normalizeName(item.Show.Network.Name)

	}

	if item.Show.WebChannel != nil && item.Show.WebChannel.Name != "" {

		return normalizeName(item.Show.WebChannel.Name)

	}

	return ""

}

func programFromItem(item tvmazeItem) (Program, bool) {

	if item.Runtime <= 0 {

		item.Runtime = 30

	}

	start, err := time.Parse(time.RFC3339, item.AirStamp)

	if err != nil {

		return Program{}, false

	}

	title := strings.TrimSpace(item.Show.Name)

	if title == "" {

		return Program{}, false

	}

	summary := stripHTML(item.Summary)

	if summary == "" {

		summary = stripHTML(item.Show.Summary)

	}

	image := ""

	if item.Show.Image != nil {

		image = item.Show.Image.Medium

		if image == "" {

			image = item.Show.Image.Original

		}

	}

	rating := ""

	if item.Show.Rating != nil && item.Show.Rating.Average != nil {

		rating = fmt.Sprintf("%.1f", *item.Show.Rating.Average)

	}

	return Program{

		Title: title,
		EpisodeTitle: strings.TrimSpace(item.Name),
		Summary: summary,

		StartsAt: start,
		Runtime: item.Runtime,

		Image: image,
		Season: item.Season,
		Episode: item.Number,

		Genres: append([]string(nil), item.Show.Genres...),
		Rating: rating,
		Network: networkKey(item),

	}, true

}

func programsForChannel(ch catalog.Channel, byNetwork map[string][]Program) []Program {

	candidates := []string{
		ch.Name,
		ch.Network,
	}

	candidates = append(candidates, ch.AltNames...)

	for _, c := range candidates {

		key := normalizeName(c)

		if key == "" {

			continue

		}

		if progs, ok := byNetwork[key]; ok {

			return progs

		}

		// Prefix / containment for names like "FOX" vs "Fox Broadcasting Company".
		for net, progs := range byNetwork {

			if len(key) >= 3 && (strings.HasPrefix(net, key) || strings.HasPrefix(key, net)) {

				return progs

			}

		}

	}

	return nil

}

func splitPrograms(programs []Program, now time.Time) (current, next *Program, upcoming []Program) {

	var nextStart time.Time

	for i := range programs {

		p := programs[i]
		end := p.StartsAt.Add(time.Duration(p.Runtime) * time.Minute)

		if !now.Before(p.StartsAt) && now.Before(end) {

			cp := p
			current = &cp

		} else if p.StartsAt.After(now) {

			if next == nil || p.StartsAt.Before(nextStart) {

				np := p
				next = &np
				nextStart = p.StartsAt

			}

			if len(upcoming) < 6 {

				upcoming = append(upcoming, p)

			}

		}

	}

	return current, next, upcoming

}

func normalizeName(name string) string {

	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimSuffix(name, " usa")
	name = strings.TrimSuffix(name, " us")
	name = strings.TrimSuffix(name, " hd")
	name = strings.TrimSuffix(name, " fhd")
	name = strings.TrimSuffix(name, " uhd")

	if i := strings.Index(name, " ("); i >= 0 {

		name = strings.TrimSpace(name[:i])

	}

	return name

}

func stripHTML(s string) string {

	if s == "" {

		return ""

	}

	var b strings.Builder
	inTag := false

	for _, r := range s {

		switch {

		case r == '<':

			inTag = true

		case r == '>':

			inTag = false

		case !inTag:

			b.WriteRune(r)

		}

	}

	return strings.TrimSpace(b.String())

}
