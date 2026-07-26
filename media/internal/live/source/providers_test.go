package source_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"mediakit/internal/live/source"
)

func TestDefaultProvidersRegistered(t *testing.T) {

	r := source.Default()
	names := r.Providers()

	want := map[string]bool{"daddylive": true, "ntv": true, "pluto": true, "iptvorg": true}

	if len(names) != len(want) {

		t.Fatalf("providers=%v want 4", names)

	}

	for _, n := range names {

		if !want[n] {

			t.Fatalf("unexpected provider %q", n)

		}

	}

	pubs := r.PublicProviders()

	if len(pubs) < 2 || pubs[0].Key != "auto" {

		t.Fatalf("public providers missing auto: %+v", pubs)

	}

	for _, p := range pubs {

		if strings.Contains(strings.ToLower(p.Label), "daddy") ||
			strings.Contains(strings.ToLower(p.Label), "ntv") ||
			strings.Contains(strings.ToLower(p.Label), "pluto") ||
			strings.Contains(strings.ToLower(p.Key), "daddy") {

			t.Fatalf("public provider not anonymized: %+v", p)

		}

	}

}

func TestDaddyLiveResolveESPN(t *testing.T) {

	if testing.Short() {

		t.Skip("network")

	}

	p := source.NewDaddyLive()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := p.Resolve(ctx, source.Request{Name: "ESPN", Country: "us"})

	if err != nil {

		t.Fatalf("resolve: %v", err)

	}

	assertPlayable(t, stream, "daddylive") // direct Provider still sets internal name

}

func TestDaddyLiveResolveCNN(t *testing.T) {

	if testing.Short() {

		t.Skip("network")

	}

	p := source.NewDaddyLive()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := p.Resolve(ctx, source.Request{Name: "CNN", Country: "us"})

	if err != nil {

		t.Fatalf("resolve: %v", err)

	}

	assertPlayable(t, stream, "daddylive")

}

func TestPlutoResolve(t *testing.T) {

	if testing.Short() {

		t.Skip("network")

	}

	p := source.NewPluto()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Pluto brand channels should match strongly.
	stream, err := p.Resolve(ctx, source.Request{Name: "Pluto TV Trending Now"})

	if err != nil {

		// Fallback: try a broader known FAST brand if lineup renamed.
		stream, err = p.Resolve(ctx, source.Request{Name: "Pluto TV Spotlight"})

	}

	if err != nil {

		t.Fatalf("resolve: %v", err)

	}

	assertPlayable(t, stream, "pluto")

}

func TestIPTVOrgResolveByChannelID(t *testing.T) {

	if testing.Short() {

		t.Skip("network")

	}

	p := source.NewIPTVOrg()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Stable free religious network with reliable HLS on iptv-org.
	stream, err := p.Resolve(ctx, source.Request{
		ChannelID: "3ABNEnglish.us",
		Name: "3ABN English",
		Country: "us",
	})

	if err != nil {

		// Fall back to another commonly-listed free stream.
		stream, err = p.Resolve(ctx, source.Request{
			ChannelID: "00sReplay.us",
			Name: "00s Replay",
			Country: "us",
		})

	}

	if err != nil {

		t.Fatalf("resolve: %v", err)

	}

	assertPlayable(t, stream, "iptvorg")

}

func TestNTVResolveOrSkip(t *testing.T) {

	if testing.Short() {

		t.Skip("network")

	}

	p := source.NewNTV()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := p.Resolve(ctx, source.Request{Name: "CNN", Country: "us"})

	if err != nil {

		// NTV's cdnlive edge frequently 502s — provider stays registered but
		// is allowed to fail closed so other sources can win.
		t.Logf("ntv not playable right now (expected intermittently): %v", err)
		return

	}

	assertPlayable(t, stream, "ntv")

}

func TestResolverFallbackChain(t *testing.T) {

	if testing.Short() {

		t.Skip("network")

	}

	r := source.Default()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Major US channel — DaddyLive should usually win.
	stream, err := r.Resolve(ctx, source.Request{Name: "ESPN", Country: "us"})

	if err != nil {

		t.Fatalf("chain resolve ESPN: %v", err)

	}

	if stream.URL == "" || !stream.IsHLS {

		t.Fatalf("bad stream: %+v", stream)

	}

	if stream.Provider == "" {

		t.Fatalf("missing provider on stream")

	}

	// Resolver must only expose anonymized public keys.
	if stream.Provider != "s1" && stream.Provider != "s2" && stream.Provider != "s3" && stream.Provider != "s4" {

		t.Fatalf("expected public provider key, got %q", stream.Provider)

	}

	t.Logf("ESPN resolved via %s", stream.Provider)

	// Second channel for coverage.
	stream2, err := r.Resolve(ctx, source.Request{Name: "ABC", Country: "us"})

	if err != nil {

		t.Fatalf("chain resolve ABC: %v", err)

	}

	t.Logf("ABC resolved via %s url=%s", stream2.Provider, truncate(stream2.URL, 80))

	// Manual provider selection by public key.
	stream3, err := r.ResolveWith(ctx, source.Request{Name: "ESPN", Country: "us"}, "s1")

	if err != nil {

		t.Fatalf("resolve with s1: %v", err)

	}

	if stream3.Provider != "s1" {

		t.Fatalf("want provider s1, got %q", stream3.Provider)

	}

}

func assertPlayable(t *testing.T, stream source.Stream, wantProvider string) {

	t.Helper()

	if stream.URL == "" {

		t.Fatal("empty url")

	}

	if !stream.IsHLS && !strings.Contains(strings.ToLower(stream.URL), "m3u8") {

		t.Fatalf("expected hls, got %s", stream.URL)

	}

	if stream.Provider != "" && stream.Provider != wantProvider {

		t.Fatalf("provider=%q want %q", stream.Provider, wantProvider)

	}

	if !strings.HasPrefix(stream.URL, "http") {

		t.Fatalf("url not http: %s", stream.URL)

	}

	t.Logf("%s ok url=%s headers=%v", wantProvider, truncate(stream.URL, 90), stream.Headers)

}

func truncate(s string, n int) string {

	if len(s) <= n {

		return s

	}

	return s[:n] + "..."

}
