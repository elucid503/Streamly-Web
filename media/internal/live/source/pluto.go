package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Pluto TV — free ad-supported live channels (FMHY Live TV list).
// Official boot + guide APIs; HLS via stitcher.

type plutoProvider struct {

	client *http.Client

	mu sync.Mutex
	sessionToken string
	stitcher string
	stitcherParams string
	channels []plutoChannel
	sessionAt time.Time
	channelsAt time.Time

}

type plutoBoot struct {

	SessionToken string `json:"sessionToken"`
	StitcherParams string `json:"stitcherParams"`
	Servers struct {

		Stitcher string `json:"stitcher"`

	} `json:"servers"`

}

type plutoGuideResponse struct {

	Data []plutoChannel `json:"data"`

}

type plutoChannel struct {

	ID string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Stitched *struct {

		Path string `json:"path"`

	} `json:"stitched"`

}

// NewPluto builds the Pluto TV source provider.
func NewPluto() Provider {

	return &plutoProvider{

		client: newHTTPClient(20 * time.Second),

	}

}

func (p *plutoProvider) Name() string {

	return "pluto"

}

func (p *plutoProvider) Resolve(ctx context.Context, req Request) (Stream, error) {

	if err := p.ensureSession(ctx); err != nil {

		return Stream{}, err

	}

	if err := p.ensureChannels(ctx); err != nil {

		return Stream{}, err

	}

	ch, ok := p.match(req)

	if !ok {

		return Stream{}, fmt.Errorf("pluto: no channel match for %q", req.Name)

	}

	path := "/stitch/hls/channel/" + ch.ID + "/master.m3u8"

	if ch.Stitched != nil && ch.Stitched.Path != "" {

		path = ch.Stitched.Path

	}

	streamURL := strings.TrimRight(p.stitcher, "/") + path

	if p.stitcherParams != "" {

		if strings.Contains(streamURL, "?") {

			streamURL += "&" + p.stitcherParams

		} else {

			streamURL += "?" + p.stitcherParams

		}

	}

	// Pluto stitcher allows anonymous browser playback (CORS). Avoid attaching
	// Referer/UA so the app can play direct when proxyLiveStreams is off.
	if !verifyPlaylist(ctx, p.client, streamURL, nil) {

		return Stream{}, fmt.Errorf("pluto: playlist not playable for %q", ch.Name)

	}

	return Stream{

		URL: streamURL,
		IsHLS: true,
		Provider: p.Name(),

	}, nil

}

func (p *plutoProvider) match(req Request) (plutoChannel, bool) {

	p.mu.Lock()
	defer p.mu.Unlock()

	bestIdx := -1
	bestScore := 0

	for i, ch := range p.channels {

		score := matchScore(req, ch.Name)

		// Slug sometimes matches better: "cnn" vs "CNN".
		if s := matchScore(req, strings.ReplaceAll(ch.Slug, "-", " ")); s > score {

			score = s

		}

		if score > bestScore {

			bestScore = score
			bestIdx = i

		}

	}

	// Pluto is mostly FAST rebrands — require a stronger name match.
	if bestIdx < 0 || bestScore < 85 {

		return plutoChannel{}, false

	}

	return p.channels[bestIdx], true

}

func (p *plutoProvider) ensureSession(ctx context.Context) error {

	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.sessionAt) < 50*time.Minute && p.sessionToken != "" && p.stitcher != "" {

		return nil

	}

	clientID := fmt.Sprintf("streamly-%d", time.Now().UnixNano())

	q := url.Values{}
	q.Set("appName", "web")
	q.Set("appVersion", "9.0.0")
	q.Set("deviceVersion", "9.0.0")
	q.Set("deviceModel", "web")
	q.Set("deviceMake", "Chrome")
	q.Set("deviceType", "web")
	q.Set("clientID", clientID)
	q.Set("clientModelNumber", "1.0")
	q.Set("serverSideAds", "false")
	q.Set("constraints", "")

	bootURL := "https://boot.pluto.tv/v4/start?" + q.Encode()

	body, status, err := getText(ctx, p.client, bootURL, map[string]string{
		"Accept": "application/json",
	})

	if err != nil {

		return fmt.Errorf("pluto: boot: %w", err)

	}

	if status != http.StatusOK {

		return fmt.Errorf("pluto: boot status %d", status)

	}

	var boot plutoBoot

	if err := json.Unmarshal([]byte(body), &boot); err != nil {

		return fmt.Errorf("pluto: boot decode: %w", err)

	}

	if boot.SessionToken == "" || boot.Servers.Stitcher == "" {

		return fmt.Errorf("pluto: incomplete boot session")

	}

	p.sessionToken = boot.SessionToken
	p.stitcher = boot.Servers.Stitcher
	p.stitcherParams = boot.StitcherParams
	p.sessionAt = time.Now()

	return nil

}

func (p *plutoProvider) ensureChannels(ctx context.Context) error {

	p.mu.Lock()

	if time.Since(p.channelsAt) < 30*time.Minute && len(p.channels) > 0 {

		p.mu.Unlock()
		return nil

	}

	token := p.sessionToken
	p.mu.Unlock()

	if token == "" {

		return fmt.Errorf("pluto: missing session")

	}

	// Paginate — Pluto has hundreds of channels.
	var all []plutoChannel

	for offset := 0; offset < 1000; offset += 100 {

		u := fmt.Sprintf("https://service-channels.clusters.pluto.tv/v2/guide/channels?channelIds=&offset=%d&limit=100&sort=number%%3Aasc", offset)

		body, status, err := getText(ctx, p.client, u, map[string]string{
			"Accept": "application/json",
			"Authorization": "Bearer " + token,
		})

		if err != nil {

			return fmt.Errorf("pluto: channels: %w", err)

		}

		if status != http.StatusOK {

			return fmt.Errorf("pluto: channels status %d", status)

		}

		var parsed plutoGuideResponse

		if err := json.Unmarshal([]byte(body), &parsed); err != nil {

			return fmt.Errorf("pluto: channels decode: %w", err)

		}

		if len(parsed.Data) == 0 {

			break

		}

		all = append(all, parsed.Data...)

		if len(parsed.Data) < 100 {

			break

		}

	}

	if len(all) == 0 {

		return fmt.Errorf("pluto: empty channel guide")

	}

	p.mu.Lock()
	p.channels = all
	p.channelsAt = time.Now()
	p.mu.Unlock()

	return nil

}
