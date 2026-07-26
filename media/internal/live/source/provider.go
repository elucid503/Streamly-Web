package source

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnavailable means no source provider could resolve a playable stream.
var ErrUnavailable = errors.New("live/source: no stream available")

// ErrNoProviders means the source layer has no registered providers.
var ErrNoProviders = errors.New("live/source: no providers configured")

// Request carries catalog channel identity for source resolution.
// Providers match by ID, name, alt names, or network — never by stored stream URLs.
type Request struct {

	ChannelID string
	Name string
	AltNames []string
	Network string
	Country string

}

// Stream is a playable media reference returned by a source provider.
type Stream struct {

	URL string
	IsHLS bool
	Headers map[string]string
	Provider string

}

// Provider resolves catalog channel identity into playable streams.
type Provider interface {

	// Name is a short stable id (e.g. "ntv", "daddylive").
	Name() string

	// Resolve returns a stream for the given catalog channel request.
	Resolve(ctx context.Context, req Request) (Stream, error)

}

// Resolver queries registered providers in order until one succeeds.
type Resolver struct {

	providers []Provider

}

// NewResolver builds a multi-provider resolver.
func NewResolver(providers ...Provider) *Resolver {

	out := make([]Provider, 0, len(providers))

	for _, p := range providers {

		if p != nil {

			out = append(out, p)

		}

	}

	return &Resolver{providers: out}

}

// Default builds the production provider chain evaluated from FMHY Live TV options.
// Order prefers currently-reliable US cable sources first, then free/public indexes.
func Default() *Resolver {

	return NewResolver(
		NewDaddyLive(), // FMHY ⭐ — best major US cable coverage right now
		NewNTV(),       // FMHY ⭐ — kept; fails closed when cdnlive 502s
		NewPluto(),     // FMHY free FAST — reliable official API
		NewIPTVOrg(),   // EasyWebTV / iptv-org open streams (catalog ID match)
	)

}

// Resolve walks providers for a stream (automatic order).
func (r *Resolver) Resolve(ctx context.Context, req Request) (Stream, error) {

	return r.ResolveWith(ctx, req, "")

}

// ResolveWith resolves using a specific public provider key, or auto when key is empty/"auto".
// The returned Stream.Provider is always the public anonymized key (never internal names).
func (r *Resolver) ResolveWith(ctx context.Context, req Request, publicKey string) (Stream, error) {

	if r == nil || len(r.providers) == 0 {

		return Stream{}, ErrNoProviders

	}

	if ctx == nil {

		ctx = context.Background()

	}

	publicKey = normalizePublicKey(publicKey)

	providers := r.providers

	if publicKey != "" && publicKey != "auto" {

		internal := InternalName(publicKey)

		if internal == "" {

			return Stream{}, fmt.Errorf("live/source: unknown provider %q", publicKey)

		}

		p := r.providerByName(internal)

		if p == nil {

			return Stream{}, fmt.Errorf("live/source: provider unavailable")

		}

		providers = []Provider{p}

	}

	var last error

	for _, p := range providers {

		stream, err := p.Resolve(ctx, req)

		if err != nil {

			last = err
			continue

		}

		if stream.URL == "" {

			continue

		}

		stream.Provider = PublicKey(p.Name())

		if stream.Provider == "" {

			stream.Provider = "auto"

		}

		if !stream.IsHLS && looksLikeHLS(stream.URL) {

			stream.IsHLS = true

		}

		return stream, nil

	}

	if last != nil {

		return Stream{}, last

	}

	return Stream{}, ErrUnavailable

}

// Providers returns registered internal provider names (for tests/ops only).
func (r *Resolver) Providers() []string {

	if r == nil {

		return nil

	}

	names := make([]string, 0, len(r.providers))

	for _, p := range r.providers {

		names = append(names, p.Name())

	}

	return names

}

// PublicProviders returns anonymized provider options for clients.
func (r *Resolver) PublicProviders() []PublicProvider {

	return PublicProviderList(r)

}

func (r *Resolver) providerByName(name string) Provider {

	for _, p := range r.providers {

		if p.Name() == name {

			return p

		}

	}

	return nil

}

func looksLikeHLS(raw string) bool {

	lower := toLower(raw)

	return contains(lower, ".m3u8") || contains(lower, "m3u8?")

}

func toLower(s string) string {

	b := make([]byte, len(s))

	for i := 0; i < len(s); i++ {

		c := s[i]

		if c >= 'A' && c <= 'Z' {

			c += 'a' - 'A'

		}

		b[i] = c

	}

	return string(b)

}

func contains(s, sub string) bool {

	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)

}

func indexOf(s, sub string) int {

	for i := 0; i+len(sub) <= len(s); i++ {

		if s[i:i+len(sub)] == sub {

			return i

		}

	}

	return -1

}
