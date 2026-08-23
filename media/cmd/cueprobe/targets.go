package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"mediakit/internal/live/source"
)

type targetKind string

const (
	kindSkip targetKind = "skip"
	kindStreamly targetKind = "streamly"
	kindXumo targetKind = "xumo"
	kindDistro targetKind = "distro"
	kindScrape targetKind = "scrape"
)

type target struct {

	Name string
	FMHY string
	Kind targetKind
	Note string

	Provider source.Provider
	ChannelNames []string
	ChannelIDs []string
	PageURL string
	XumoID string

}

func fmhyLiveTVTargets() []target {

	return []target{

		{
			Name: "TVCL",
			FMHY: "https://www.tvchannellists.com/",
			Kind: kindSkip,
			Note: "channel encyclopedia, no streams",
		},
		{
			Name: "NTV",
			FMHY: "https://ntv.cx/",
			Kind: kindStreamly,
			Provider: source.NewNTV(),
			ChannelNames: []string{"ESPN", "CNN", "NBC"},
		},
		{
			Name: "StreamSports99",
			FMHY: "https://streamsports99.website/",
			Kind: kindScrape,
			PageURL: "https://streamsports99.website/",
		},
		{
			Name: "DaddyLive TV",
			FMHY: "https://dlhd.st/",
			Kind: kindStreamly,
			Provider: source.NewDaddyLive(),
			ChannelNames: []string{"ESPN", "CNN", "NBC"},
		},
		{
			Name: "Famelack",
			FMHY: "https://famelack.com/",
			Kind: kindScrape,
			PageURL: "https://famelack.com/",
		},
		{
			Name: "EasyWebTV / iptv-org",
			FMHY: "https://zhangboheng.github.io/Easy-Web-TV-M3u8/routes/tv.html",
			Kind: kindStreamly,
			Provider: source.NewIPTVOrg(),
			ChannelIDs: []string{"BloombergTV.us", "NASA.tv", "C-SPAN.us", "3ABNEnglish.us"},
			ChannelNames: []string{"Bloomberg", "NASA", "C-SPAN"},
		},
		{
			Name: "SportsBite TV",
			FMHY: "https://sportsbite.org/channels",
			Kind: kindScrape,
			PageURL: "https://sportsbite.org/channels",
		},
		{
			Name: "TitanTV",
			FMHY: "https://titantv.com/",
			Kind: kindSkip,
			Note: "program listings only",
		},
		{
			Name: "TV Freedom",
			FMHY: "https://tvfreedom.surge.sh/",
			Kind: kindScrape,
			PageURL: "https://tvfreedom.surge.sh/",
		},
		{
			Name: "vavoo.to",
			FMHY: "https://vavoo.to/",
			Kind: kindScrape,
			PageURL: "https://vavoo.to/",
		},
		{
			Name: "Live24",
			FMHY: "https://livelive24.com/",
			Kind: kindScrape,
			PageURL: "https://livelive24.com/",
		},
		{
			Name: "Xumo Play (FAST loop)",
			FMHY: "https://play.xumo.com/networks",
			Kind: kindXumo,
			XumoID: "99951252",
			Note: "Cheaters — scheduled FAST assets",
		},
		{
			Name: "Xumo Play (sports live flag)",
			FMHY: "https://play.xumo.com/networks",
			Kind: kindXumo,
			XumoID: "99991196",
			Note: "FOX Sports — live:true unbounded asset",
		},
		{
			Name: "Pluto TV",
			FMHY: "https://pluto.tv/live-tv",
			Kind: kindStreamly,
			Provider: source.NewPluto(),
			ChannelNames: []string{"Court TV", "ION", "Pluto TV Trending Now", "Pluto TV Spotlight"},
		},
		{
			Name: "Watchott Live",
			FMHY: "https://iptv.watchott.org/",
			Kind: kindScrape,
			PageURL: "https://iptv.watchott.org/",
		},
		{
			Name: "HOOFOOT IPTV",
			FMHY: "https://hoofoot.ru/iptv/",
			Kind: kindScrape,
			PageURL: "https://hoofoot.ru/iptv/",
		},
		{
			Name: "TV Explorer",
			FMHY: "https://tvexplorer.live/",
			Kind: kindScrape,
			PageURL: "https://tvexplorer.live/",
		},
		{
			Name: "Rive IPTV",
			FMHY: "https://www.rivestream.app/iptv",
			Kind: kindScrape,
			PageURL: "https://www.rivestream.app/iptv",
		},
		{
			Name: "FreeInterTV",
			FMHY: "http://www.freeintertv.com/",
			Kind: kindScrape,
			PageURL: "http://www.freeintertv.com/",
		},
		{
			Name: "Global Free TV",
			FMHY: "https://www.globalfreetv.com/",
			Kind: kindScrape,
			PageURL: "https://www.globalfreetv.com/",
		},
		{
			Name: "SquidTV",
			FMHY: "https://www.squidtv.net/",
			Kind: kindScrape,
			PageURL: "https://www.squidtv.net/",
		},
		{
			Name: "TVAtlas",
			FMHY: "https://tvatlas.app/",
			Kind: kindScrape,
			PageURL: "https://tvatlas.app/country/us",
		},
		{
			Name: "DistroTV",
			FMHY: "https://distro.tv/",
			Kind: kindDistro,
		},
		{
			Name: "Puffer",
			FMHY: "https://puffer.stanford.edu/",
			Kind: kindSkip,
			Note: "OTA research, signup required",
		},

	}

}

type resolvedStream struct {

	URL string
	Headers map[string]string
	Detail string

}

func resolveTarget(ctx context.Context, h *inspectHTTP, t target, channelFlag string) (resolvedStream, error) {

	switch t.Kind {

	case kindSkip:

		return resolvedStream{}, fmt.Errorf("%s", t.Note)

	case kindStreamly:

		return resolveStreamly(ctx, t, channelFlag)

	case kindXumo:

		return resolveXumo(ctx, h, t.XumoID)

	case kindDistro:

		return resolveDistro(ctx, h)

	case kindScrape:

		return resolveScrape(ctx, h, t.PageURL)

	default:

		return resolvedStream{}, fmt.Errorf("unknown kind")

	}

}

func resolveStreamly(ctx context.Context, t target, channelFlag string) (resolvedStream, error) {

	if t.Provider == nil {

		return resolvedStream{}, fmt.Errorf("no provider")

	}

	names := append([]string{}, t.ChannelNames...)

	if s := strings.TrimSpace(channelFlag); s != "" {

		names = append([]string{s}, names...)

	}

	var last error

	for _, id := range t.ChannelIDs {

		stream, err := t.Provider.Resolve(ctx, source.Request{

			ChannelID: id,
			Name: id,
			Country: "us",

		})

		if err != nil {

			last = err
			continue

		}

		if stream.URL != "" {

			return resolvedStream{

				URL: stream.URL,
				Headers: stream.Headers,
				Detail: "id=" + id,

			}, nil

		}

	}

	for _, name := range names {

		stream, err := t.Provider.Resolve(ctx, source.Request{

			Name: name,
			Country: "us",

		})

		if err != nil {

			last = err
			continue

		}

		if stream.URL != "" {

			return resolvedStream{

				URL: stream.URL,
				Headers: stream.Headers,
				Detail: "ch=" + name,

			}, nil

		}

	}

	if last != nil {

		return resolvedStream{}, last

	}

	return resolvedStream{}, fmt.Errorf("no stream")

}

func resolveXumo(ctx context.Context, h *inspectHTTP, channelID string) (resolvedStream, error) {

	rawURL := "https://android-tv-mds.xumo.com/v2/channels/channel/" + channelID + "/broadcast.json?hour=0"

	body, status, err := h.getText(ctx, rawURL, map[string]string{

		"Accept": "application/json",

	})

	if err != nil {

		return resolvedStream{}, err

	}

	if status < 200 || status >= 300 {

		return resolvedStream{}, fmt.Errorf("broadcast status %d", status)

	}

	var parsed struct {

		Assets []struct {

			ID string `json:"id"`
			Live bool `json:"live"`

		} `json:"assets"`

		SSAIStreamURL string `json:"ssaiStreamUrl"`

	}

	if err := json.Unmarshal([]byte(body), &parsed); err != nil {

		return resolvedStream{}, err

	}

	streamURL := parsed.SSAIStreamURL

	if streamURL == "" {

		return resolvedStream{}, fmt.Errorf("no ssaiStreamUrl (assets=%d)", len(parsed.Assets))

	}

	streamURL = fillXumoMacros(streamURL)

	live := false

	if len(parsed.Assets) == 1 {

		live = parsed.Assets[0].Live

	}

	return resolvedStream{

		URL: streamURL,
		Headers: map[string]string{

			"Referer": "https://play.xumo.com/",
			"Origin": "https://play.xumo.com",

		},
		Detail: fmt.Sprintf("id=%s live=%t assets=%d", channelID, live, len(parsed.Assets)),

	}, nil

}

func fillXumoMacros(raw string) string {

	repl := strings.NewReplacer(
		"[PLATFORM]", "web",
		"[IFA_TYPE]", "dpid",
		"[IFA]", "00000000-0000-0000-0000-000000000000",
		"[AMZN_APP_ID]", "",
		"[LAT]", "0",
		"[LON]", "0",
		"[OS]", "web",
		"[OS_VERSION]", "1",
		"[IS_LAT]", "1",
		"[CCPA_Value]", "1---",
		"[IAB_content_category]", "",
		"[content_language]", "en",
		"[content_rating]", "",
		"[device_make]", "Chrome",
		"[device_model]", "web",
		"[publica_site_id]", "",
		"[APP_VERSION]", "1.0",
		"[app_bundle]", "",
		"[app_store_url]", "",
	)

	return repl.Replace(raw)

}

func resolveDistro(ctx context.Context, h *inspectHTTP) (resolvedStream, error) {

	body, status, err := h.getText(ctx, "https://tv.jsrdn.com/tv_v5/getfeed.php?type=live", nil)

	if err != nil {

		return resolvedStream{}, err

	}

	if status < 200 || status >= 300 {

		return resolvedStream{}, fmt.Errorf("feed status %d", status)

	}

	urls := extractM3U8s(body)

	if len(urls) == 0 {

		return resolvedStream{}, fmt.Errorf("no m3u8 in Distro feed")

	}

	// Prefer Distro-hosted playlists; fall back to the first live URL.
	chosen := urls[0]

	for _, u := range urls {

		if strings.Contains(u, "jsrdn.com") || strings.Contains(u, "distro") {

			chosen = u
			break

		}

	}

	chosen = stripDistroMacros(chosen)

	return resolvedStream{

		URL: chosen,
		Detail: fmt.Sprintf("feed m3u8s=%d", len(urls)),

	}, nil

}

func stripDistroMacros(raw string) string {

	u, err := url.Parse(raw)

	if err != nil {

		return raw

	}

	q := u.Query()

	for k := range q {

		if strings.HasPrefix(k, "ads.") {

			q.Del(k)

		}

	}

	u.RawQuery = q.Encode()

	return u.String()

}

func resolveScrape(ctx context.Context, h *inspectHTTP, pageURL string) (resolvedStream, error) {

	body, status, err := h.getText(ctx, pageURL, map[string]string{

		"Accept": "text/html,application/json,*/*",
		"Referer": pageURL,

	})

	if err != nil {

		return resolvedStream{}, err

	}

	if status < 200 || status >= 300 {

		return resolvedStream{}, fmt.Errorf("page status %d", status)

	}

	urls := extractM3U8s(body)

	if len(urls) == 0 {

		return resolvedStream{}, fmt.Errorf("no m3u8 in HTML (likely JS-rendered)")

	}

	return resolvedStream{

		URL: urls[0],
		Headers: map[string]string{

			"Referer": pageURL,
			"Origin": originOf(pageURL),

		},
		Detail: fmt.Sprintf("scraped m3u8s=%d", len(urls)),

	}, nil

}

func originOf(raw string) string {

	u, err := url.Parse(raw)

	if err != nil || u.Scheme == "" || u.Host == "" {

		return ""

	}

	return u.Scheme + "://" + u.Host

}
