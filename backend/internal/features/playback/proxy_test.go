package playback

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"streamly/internal/config"

	"github.com/gin-gonic/gin"
)

func TestProxyPreservesHeadersOnPlaylistChildren(t *testing.T) {

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/live.m3u8" {

			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			fmt.Fprint(w, "#EXTM3U\n#EXTINF:4,\nsegment.ts\n")
			return

		}

		if r.Header.Get("Referer") != "https://player.example/" || r.Header.Get("Origin") != "https://player.example" {

			w.WriteHeader(http.StatusForbidden)
			return

		}

		w.Header().Set("Content-Type", "video/mp2t")
		fmt.Fprint(w, "segment payload")

	}))
	defer upstream.Close()

	service := NewProxyService(&config.Config{ProxyTokenTTL: time.Hour})
	session, err := service.CreateSessionWithHeaders(context.Background(), upstream.URL+"/live.m3u8", map[string]string{

		"Referer": "https://player.example/",
		"Origin": "https://player.example",

	}, true)

	if err != nil {

		t.Fatal(err)

	}

	router := gin.New()
	router.GET("/api/proxy/:token", NewProxyHandler(service).Serve)
	proxy := httptest.NewServer(router)
	defer proxy.Close()

	response, err := http.Get(proxy.URL + session.ProxyPath)

	if err != nil {

		t.Fatal(err)

	}

	body, err := io.ReadAll(response.Body)
	response.Body.Close()

	if err != nil || response.StatusCode != http.StatusOK {

		t.Fatalf("playlist status=%d err=%v", response.StatusCode, err)

	}

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	response, err = http.Get(lines[len(lines)-1])

	if err != nil {

		t.Fatal(err)

	}

	body, err = io.ReadAll(response.Body)
	response.Body.Close()

	if err != nil || response.StatusCode != http.StatusOK || string(body) != "segment payload" {

		t.Fatalf("segment status=%d body=%q err=%v", response.StatusCode, body, err)

	}

}
