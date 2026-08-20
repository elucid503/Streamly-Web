package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultChannelsURL = "https://iptv-org.github.io/api/channels.json"
	defaultLogosURL    = "https://iptv-org.github.io/api/logos.json"

	fetchTimeout = 45 * time.Second

	browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
)

// iptv-org is the open channel metadata index used as the primary catalog source.
// TV Channel Lists / TitanTV do not expose a free public API suitable for
// production; iptv-org provides stable IDs, logos, owners, networks, and categories.

type iptvChannel struct {

	ID string `json:"id"`
	Name string `json:"name"`
	AltNames []string `json:"alt_names"`
	Network *string `json:"network"`
	Owners []string `json:"owners"`
	Country string `json:"country"`
	Categories []string `json:"categories"`
	IsNSFW bool `json:"is_nsfw"`
	Launched *string `json:"launched"`
	Closed *string `json:"closed"`
	ReplacedBy *string `json:"replaced_by"`
	Website *string `json:"website"`

}

type iptvLogo struct {

	Channel string `json:"channel"`
	InUse bool `json:"in_use"`
	Width int `json:"width"`
	Height int `json:"height"`
	Format string `json:"format"`
	URL string `json:"url"`

}

func channelsURL() string {

	if v := strings.TrimSpace(os.Getenv("TV_CHANNEL_METADATA_URL")); v != "" {

		return v

	}

	return defaultChannelsURL

}

func logosURL() string {

	if v := strings.TrimSpace(os.Getenv("TV_CHANNEL_LOGOS_URL")); v != "" {

		return v

	}

	return defaultLogosURL

}

func fetchIPTVCatalog(httpClient *http.Client) ([]Channel, error) {

	if httpClient == nil {

		httpClient = &http.Client{Timeout: fetchTimeout}

	}

	rawChannels, err := fetchJSON(httpClient, channelsURL())

	if err != nil {

		return nil, fmt.Errorf("catalog: channels: %w", err)

	}

	var channels []iptvChannel

	if err := json.Unmarshal(rawChannels, &channels); err != nil {

		return nil, fmt.Errorf("catalog: channels decode: %w", err)

	}

	rawLogos, err := fetchJSON(httpClient, logosURL())

	if err != nil {

		return nil, fmt.Errorf("catalog: logos: %w", err)

	}

	var logos []iptvLogo

	if err := json.Unmarshal(rawLogos, &logos); err != nil {

		return nil, fmt.Errorf("catalog: logos decode: %w", err)

	}

	logoByChannel := bestLogosByChannel(logos)

	out := make([]Channel, 0, 512)

	for _, raw := range channels {

		if !includeIPTVChannel(raw) {

			continue

		}

		logo := logoByChannel[raw.ID]

		// Require artwork for a polished catalog UI. Team RSNs may still
		// enter without a logo so sports matching can find them.
		if logo.URL == "" && !isTeamSportsChannel(raw) {

			continue

		}

		ch := channelFromIPTV(raw, logo.URL)
		out = append(out, ch)

	}

	return dedupeByName(out), nil

}

func includeIPTVChannel(raw iptvChannel) bool {

	if raw.IsNSFW || raw.ID == "" || strings.TrimSpace(raw.Name) == "" {

		return false

	}

	if raw.Closed != nil && strings.TrimSpace(*raw.Closed) != "" {

		return false

	}

	if raw.ReplacedBy != nil && strings.TrimSpace(*raw.ReplacedBy) != "" {

		return false

	}

	country := strings.ToUpper(strings.TrimSpace(raw.Country))

	// Product catalog is US (+ CA) majors plus team RSNs, not the full
	// iptv-org dump (local subchannels, FAST farms, etc.).
	if country != "US" && country != "CA" {

		return false

	}

	network := ""

	if raw.Network != nil {

		network = strings.ToLower(strings.TrimSpace(*raw.Network))

	}

	// Curated majors always pass (ESPN, CNN, HBO, …).
	if isMajorName(raw.Name) {

		return true

	}

	// Team-specific RSNs (YES Network, FanDuel Sports Network *, MSG, …)
	// even when iptv-org omits category/owner/network.
	if isTeamSportsChannel(raw) {

		return true

	}

	// Drop FAST/AVOD farms entirely for non-majors.
	if isFASTNetwork(network) {

		return false

	}

	// Beyond the major list, only keep US/CA news and sports brands with
	// real station metadata — not the long tail of local general stations.
	if !hasPriorityCategory(raw.Categories) {

		return false

	}

	return len(raw.Owners) > 0 || network != ""

}

func hasPriorityCategory(categories []string) bool {

	for _, c := range categories {

		switch strings.ToLower(strings.TrimSpace(c)) {

		case "news", "sports":

			return true

		}

	}

	return false

}

func isFASTNetwork(network string) bool {

	switch network {

	case "pluto tv", "tubi", "freevee", "xumo", "roku channel", "samsung tv plus", "lg channels":

		return true

	}

	return false

}

func channelFromIPTV(raw iptvChannel, logoURL string) Channel {

	countryCode := strings.ToLower(strings.TrimSpace(raw.Country))

	network := ""

	if raw.Network != nil {

		network = strings.TrimSpace(*raw.Network)

	}

	website := ""

	if raw.Website != nil {

		website = strings.TrimSpace(*raw.Website)

	}

	categories := append([]string(nil), raw.Categories...)

	if len(categories) == 0 && isTeamSportsChannel(raw) {

		categories = []string{"sports"}

	}

	primary := displayCategory(categories)
	altNames := mergeAltNames(raw.AltNames, extraTeamAltNames(raw.Name), raw.Name)

	return Channel{

		ID: raw.ID,
		Name: strings.TrimSpace(raw.Name),
		Slug: slugify(raw.Name),
		Code: countryCode,
		Logo: logoURL,

		Country: Country{

			Code: countryCode,
			Name: countryName(countryCode),

		},

		Category: primary,
		Categories: categories,

		Network: network,
		Owners: append([]string(nil), raw.Owners...),
		Website: website,
		AltNames: altNames,

		Enriched: logoURL != "" || primary != "" || network != "",

	}

}

func displayCategory(categories []string) string {

	if len(categories) == 0 {

		return ""

	}

	// Prefer a product-facing label order.
	priority := []string{
		"sports", "news", "movies", "series", "entertainment", "kids",
		"documentary", "music", "comedy", "lifestyle", "cooking", "travel",
	}

	lower := make(map[string]string, len(categories))

	for _, c := range categories {

		c = strings.TrimSpace(c)

		if c == "" {

			continue

		}

		lower[strings.ToLower(c)] = c

	}

	for _, p := range priority {

		if v, ok := lower[p]; ok {

			return titleCategory(v)

		}

	}

	for _, c := range categories {

		c = strings.TrimSpace(c)

		if c != "" {

			return titleCategory(c)

		}

	}

	return ""

}

func titleCategory(c string) string {

	c = strings.ReplaceAll(c, "-", " ")
	parts := strings.Fields(c)

	for i, p := range parts {

		if len(p) == 0 {

			continue

		}

		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])

	}

	return strings.Join(parts, " ")

}

func countryName(code string) string {

	switch strings.ToLower(code) {

	case "us":

		return "United States"

	case "ca":

		return "Canada"

	case "uk", "gb":

		return "United Kingdom"

	default:

		return strings.ToUpper(code)

	}

}

func bestLogosByChannel(logos []iptvLogo) map[string]iptvLogo {

	best := make(map[string]iptvLogo)

	for _, logo := range logos {

		logo.Channel = strings.TrimSpace(logo.Channel)
		logo.URL = strings.TrimSpace(logo.URL)

		if logo.Channel == "" || logo.URL == "" {

			continue

		}

		current, ok := best[logo.Channel]

		if !ok || logoScore(logo) > logoScore(current) {

			best[logo.Channel] = logo

		}

	}

	return best

}

func logoScore(logo iptvLogo) int {

	score := logo.Width * logo.Height

	if logo.InUse {

		score += 10_000_000

	}

	switch strings.ToLower(strings.TrimSpace(logo.Format)) {

	case "png":

		score += 500_000

	case "svg":

		score += 400_000

	}

	return score

}

func fetchJSON(client *http.Client, rawURL string) ([]byte, error) {

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)

	if err != nil {

		return nil, err

	}

	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))

	if err != nil {

		return nil, err

	}

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("status %d", resp.StatusCode)

	}

	return body, nil

}
