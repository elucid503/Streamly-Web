package source_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mediakit/internal/live/source"
)

type stubProvider struct {

	name string
	stream source.Stream
	err error

}

func (p stubProvider) Name() string { return p.name }

func (p stubProvider) Resolve(context.Context, source.Request) (source.Stream, error) {

	return p.stream, p.err

}

func TestResolverPreservesAllFailures(t *testing.T) {

	upstream := errors.New("TLS handshake failed")
	missing := errors.New("no matching stream")
	resolver := source.NewResolver(
		stubProvider{name: "daddylive", err: upstream},
		stubProvider{name: "iptvorg", err: missing},
	)

	_, err := resolver.Resolve(context.Background(), source.Request{ChannelID: "MASN.us"})

	for _, cause := range []error{source.ErrUnavailable, upstream, missing} {

		if !errors.Is(err, cause) {

			t.Fatalf("error %v does not preserve %v", err, cause)

		}

	}

	if !strings.Contains(err.Error(), "MASN.us") {

		t.Fatalf("missing channel context: %v", err)

	}

}

func TestResolverContinuesAfterFailure(t *testing.T) {

	resolver := source.NewResolver(
		stubProvider{name: "daddylive", err: errors.New("unavailable")},
		stubProvider{name: "iptvorg", stream: source.Stream{URL: "https://example.com/live.m3u8"}},
	)

	stream, err := resolver.Resolve(context.Background(), source.Request{ChannelID: "Example.us"})

	if err != nil || stream.URL == "" || !stream.IsHLS || stream.Provider != source.PublicKey("iptvorg") {

		t.Fatalf("fallback stream=%+v err=%v", stream, err)

	}

}

func TestResolverEmptyStreamIsUnavailable(t *testing.T) {

	resolver := source.NewResolver(stubProvider{name: "iptvorg"})
	_, err := resolver.ResolveWith(context.Background(), source.Request{ChannelID: "MASN.us"}, source.PublicKey("iptvorg"))

	if !errors.Is(err, source.ErrUnavailable) || !strings.Contains(err.Error(), "empty stream URL") {

		t.Fatalf("empty stream error=%v", err)

	}

}
