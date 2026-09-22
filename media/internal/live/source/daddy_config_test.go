package source

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func configPage(t *testing.T, config map[string]string) string {

	t.Helper()
	data, err := json.Marshal(config)

	if err != nil {

		t.Fatal(err)

	}

	encoded := base64.StdEncoding.EncodeToString(data)
	width := len(encoded) / 4
	var chunks []string

	for _, index := range []int{2, 0, 3, 1} {

		part := base64.StdEncoding.EncodeToString([]byte(encoded[index*width : (index+1)*width]))
		chunks = append(chunks, part[:3]+"X"+part[3:])

	}

	return "<script>window._econfig='" + base64.StdEncoding.EncodeToString([]byte(strings.Join(chunks, ""))) + "'</script>"

}

func TestDaddyEncodedConfig(t *testing.T) {

	for _, test := range []struct {

		name string
		config map[string]string
		want string

	}{

		{"direct", map[string]string{"stream_url_nop2p": "https://cdn.example/live.m3u8?s=token&e=123", "stream_url": "https://other.example/live.m3u8"}, "https://cdn.example/live.m3u8?s=token&e=123"},
		{"fallback", map[string]string{"stream_url": "https://cdn.example/live.m3u8"}, "https://cdn.example/live.m3u8"},
		{"invalid URL", map[string]string{"stream_url": "javascript:alert(1)"}, ""},
		{"not HLS", map[string]string{"stream_url": "https://example.com/advert.html"}, ""},

	} {

		t.Run(test.name, func(t *testing.T) {

			if got := extractDaddyPlayableURL(configPage(t, test.config)); got != test.want {

				t.Fatalf("got %q want %q", got, test.want)

			}

		})

	}

}

func TestDaddyMalformedConfig(t *testing.T) {

	for _, value := range []string{"", "x", "====", "YQ==", strings.Repeat("A", 64)} {

		if got := extractDaddyConfigURL("window._econfig='" + value + "'"); got != "" {

			t.Fatalf("malformed config returned %q", got)

		}

	}

}

func TestDaddyEncodedIframe(t *testing.T) {

	if isSkippableEmbed("https://assetrage.net/e/example") {

		t.Fatal("video player host is incorrectly classified as an ad")

	}

	page := configPage(t, map[string]string{"stream_url": "https://cdn.example/live.m3u8"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/parent" {

			fmt.Fprint(w, `<iframe src="/embed"></iframe>`)
			return

		}

		if !strings.HasSuffix(r.Referer(), "/parent") {

			t.Errorf("missing parent referrer: %q", r.Referer())

		}

		fmt.Fprint(w, page)

	}))
	defer server.Close()

	provider := &daddyLiveProvider{embedClient: server.Client()}
	stream, referrer, err := provider.resolveFromPage(context.Background(), server.URL+"/parent", server.URL, map[string]bool{})

	if err != nil || stream != "https://cdn.example/live.m3u8" || referrer != server.URL+"/" {

		t.Fatalf("stream=%q referrer=%q err=%v", stream, referrer, err)

	}

}

func TestDaddyRetainsHeadersForPublicPlaylist(t *testing.T) {

	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/live.m3u8" {

			fmt.Fprint(w, "#EXTM3U\n#EXTINF:4,\nsegment.ts\n")
			return

		}

		fmt.Fprintf(w, `streamUrl: "%s/live.m3u8"`, baseURL)

	}))
	defer server.Close()
	baseURL = server.URL

	provider := &daddyLiveProvider{

		client: server.Client(),
		embedClient: server.Client(),
		baseURL: baseURL,
		channels: map[string]int{"nbc sports bay area": 753},
		names: []string{"NBC Sports Bay Area"},
		fetchedAt: time.Now(),

	}

	stream, err := provider.Resolve(context.Background(), Request{Name: "NBC Sports Bay Area", Country: "US"})

	if err != nil {

		t.Fatal(err)

	}

	if stream.Headers["Referer"] != baseURL+"/" || stream.Headers["Origin"] != baseURL || stream.Headers["User-Agent"] != browserUA {

		t.Fatalf("lost playback headers: %v", stream.Headers)

	}

}
