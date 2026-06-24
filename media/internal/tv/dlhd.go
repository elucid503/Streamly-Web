package tv

import (
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const defaultDLHDBaseURL = "https://dlhd.pk"

var (

	iframeSrcPattern  = regexp.MustCompile(`iframe src="([^"]+)"`)
	atobSourcePattern = regexp.MustCompile(`source:\s*window\.atob\('([^']+)'\)`)
	channelCardRE     = regexp.MustCompile(`href="/watch\.php\?id=(\d+)"\s+data-title="([^"]+)"`)

)

func (c *Client) resolveDLHD(daddyID string) (ResolvedStream, error) {

	base := strings.TrimRight(dlhdBaseURL(), "/")
	streamPage := fmt.Sprintf("%s/stream/stream-%s.php", base, daddyID)

	response, err := c.get(streamPage, base+"/")

	if err != nil {

		return ResolvedStream{}, fmt.Errorf("tv: fetch dlhd stream page: %w", err)

	}

	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))

	if err != nil {

		return ResolvedStream{}, fmt.Errorf("tv: read dlhd stream page: %w", err)

	}

	if response.StatusCode != http.StatusOK {

		return ResolvedStream{}, fmt.Errorf("tv: fetch dlhd stream page: status %d", response.StatusCode)

	}

	embedURL, ok := extractIframeSrc(string(body))

	if !ok {

		return ResolvedStream{}, fmt.Errorf("tv: dlhd stream page missing embed iframe")

	}

	embedURL = normalizeURL(embedURL, base)

	embedResponse, err := c.get(embedURL, streamPage)

	if err != nil {

		return ResolvedStream{}, fmt.Errorf("tv: fetch dlhd embed: %w", err)

	}

	defer embedResponse.Body.Close()

	embedBody, err := io.ReadAll(io.LimitReader(embedResponse.Body, 2<<20))

	if err != nil {

		return ResolvedStream{}, fmt.Errorf("tv: read dlhd embed: %w", err)

	}

	if embedResponse.StatusCode != http.StatusOK {

		return ResolvedStream{}, fmt.Errorf("tv: fetch dlhd embed: status %d", embedResponse.StatusCode)

	}

	playlistURL, ok := extractAtobSource(string(embedBody))

	if !ok {

		return ResolvedStream{}, fmt.Errorf("tv: dlhd embed missing playlist source")

	}

	return ResolvedStream{

		URL: playlistURL,
		Referer: embedURL,

	}, nil

}

func dlhdBaseURL() string {

	if base := strings.TrimSpace(os.Getenv("TV_DLHD_BASE_URL")); base != "" {

		return strings.TrimRight(base, "/")

	}

	return defaultDLHDBaseURL

}

func extractIframeSrc(page string) (string, bool) {

	match := iframeSrcPattern.FindStringSubmatch(page)

	if len(match) < 2 {

		return "", false

	}

	src := strings.TrimSpace(match[1])

	if src == "" {

		return "", false

	}

	return src, true

}

func extractAtobSource(page string) (string, bool) {

	match := atobSourcePattern.FindStringSubmatch(page)

	if len(match) < 2 {

		return "", false

	}

	decoded, err := base64.StdEncoding.DecodeString(match[1])

	if err != nil {

		return "", false

	}

	playlistURL := strings.TrimSpace(string(decoded))

	if playlistURL == "" {

		return "", false

	}

	return playlistURL, true

}

func normalizeURL(raw, base string) string {

	raw = strings.TrimSpace(raw)

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {

		return raw

	}

	if strings.HasPrefix(raw, "//") {

		return "https:" + raw

	}

	if strings.HasPrefix(raw, "/") {

		return strings.TrimRight(base, "/") + raw

	}

	return raw

}

func scrapeDLHDChannels(client *http.Client) (*ChannelCatalog, error) {

	pageURL := dlhdBaseURL() + "/24-7-channels.php"

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)

	if err != nil {

		return nil, err

	}

	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", dlhdBaseURL()+"/")

	resp, err := client.Do(req)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("status %d", resp.StatusCode)

	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if err != nil {

		return nil, err

	}

	return parseDLHDChannelPage(body)

}

func parseDLHDChannelPage(body []byte) (*ChannelCatalog, error) {

	matches := channelCardRE.FindAllSubmatch(body, -1)

	if len(matches) == 0 {

		return nil, fmt.Errorf("no channel cards found")

	}

	seen := make(map[string]struct{})
	channels := make([]Channel, 0, len(matches))

	for _, m := range matches {

		daddyID := string(m[1])

		if _, ok := seen[daddyID]; ok {

			continue

		}

		seen[daddyID] = struct{}{}

		name := html.UnescapeString(string(m[2]))
		slug := slugify(name)

		channels = append(channels, Channel{

			ID:      "dl-" + daddyID,
			DaddyID: daddyID,

			Name: name,
			Slug: slug,
			Logo: dlhdBaseURL() + "/logos/" + strings.ReplaceAll(slug, "-", "_") + ".png",

			Category: "Entertainment",
			Status:   "online",
			Source:   "tv247",

		})

	}

	return &ChannelCatalog{

		Generated: time.Now().Format("2006-01-02"),
		Total:     len(channels),
		Source:    "tv247",

		Channels: channels,

	}, nil

}

func slugify(name string) string {

	name = strings.ToLower(name)
	var b strings.Builder
	lastWasSep := false

	for _, r := range name {

		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {

			b.WriteRune(r)
			lastWasSep = false

		} else if !lastWasSep && b.Len() > 0 {

			b.WriteByte('-')
			lastWasSep = true

		}

	}

	return strings.TrimRight(b.String(), "-")

}
