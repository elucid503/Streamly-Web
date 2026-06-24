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

func TestClientForTargetUsesOutboundProxyWhenConfigured(t *testing.T) {

	direct := NewProxyService(&config.Config{ProxyTokenTTL: time.Hour})

	if direct.clientForTarget("https://vixsrc.to/playlist/1.m3u8") != direct.client {

		t.Fatal("expected direct client without VIXSRC_PROXY_URL")

	}

	proxied, err := parseVixsrcProxyURL("http://127.0.0.1:9999")

	if err != nil {

		t.Fatalf("parseVixsrcProxyURL: %v", err)

	}

	withProxy := &ProxyService{

		ttl: time.Hour,
		client: direct.client,
		vixsrcProxy: proxied,
		proxiedClient: direct.client,

		tokenByKey: make(map[string]proxyTokenCacheEntry),
		entryByToken: make(map[string]ProxyEntry),

	}

	cases := []string{

		"https://vixsrc.to/playlist/1.m3u8",
		"https://sc-u5-01.vix-content.net/hls/seg.ts",

	}

	for _, target := range cases {

		if withProxy.clientForTarget(target) != withProxy.proxiedClient {

			t.Fatalf("%q: expected proxied client", target)

		}

	}

}