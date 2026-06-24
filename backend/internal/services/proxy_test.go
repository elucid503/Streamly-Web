package services

import (
	"context"
	"testing"
	"time"

	"streamly/internal/config"
)

func TestCreateSessionWithHeadersStoresOrigin(t *testing.T) {

	cfg := &config.Config{ProxyTokenTTL: time.Hour}

	proxy := NewProxyService(cfg)

	session, err := proxy.CreateSessionWithHeaders(context.Background(), "https://vixsrc.to/playlist/1.m3u8", map[string]string{

		"Referer": "https://vixsrc.to/api/movie/1",
		"Origin":  "https://vixsrc.to",

	}, true)

	if err != nil {

		t.Fatalf("CreateSessionWithHeaders: %v", err)

	}

	entry, err := proxy.ResolveToken(session.Token)

	if err != nil {

		t.Fatalf("ResolveToken: %v", err)

	}

	if entry.Referer != "https://vixsrc.to/api/movie/1" {

		t.Fatalf("unexpected referer: %q", entry.Referer)

	}

	if entry.RequestHeaders["Origin"] != "https://vixsrc.to" {

		t.Fatalf("expected origin header stored, got %+v", entry.RequestHeaders)

	}

}

func TestIsVixsrcOriginHost(t *testing.T) {

	cases := []struct {
		url  string
		want bool
	}{

		{"https://vixsrc.to/playlist/1.m3u8", true},
		{"https://vixsrc.to/api/movie/1", true},
		{"https://sc-u5-01.vix-content.net/hls/seg.ts", false},
		{"https://cdn.example.com/vixsrc-like/path.ts", false},

	}

	for _, tc := range cases {

		if got := isVixsrcOriginHost(tc.url); got != tc.want {

			t.Fatalf("%q: got %v want %v", tc.url, got, tc.want)

		}

	}

}