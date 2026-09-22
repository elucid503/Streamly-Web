package mediakit

import (
	"mediakit/internal/client"
)

// Client is the entry point for catalogue search, VOD browsing, and live TV.
type Client = client.Client

// Option configures a Client.
type Option = client.Option

// New builds a Client with optional configuration.
func New(opts ...Option) *Client {

	return client.New(opts...)

}

// WithChildMode sets the Showbox child-mode flag.
func WithChildMode(mode string) Option {

	return client.WithChildMode(mode)

}

// WithFebboxCookie sets the Febbox `ui` auth cookie required for quality links.
func WithFebboxCookie(cookie string) Option {

	return client.WithFebboxCookie(cookie)

}

// WithIntroDBKey sets an optional TheIntroDB API key.
func WithIntroDBKey(key string) Option {

	return client.WithIntroDBKey(key)

}

// WithTMDBAPIKey sets the TMDB v3 API key for episode and title metadata.
func WithTMDBAPIKey(key string) Option {

	return client.WithTMDBAPIKey(key)

}

// WithIntroCache enables a 6-hour in-memory cache for TheIntroDB lookups.
func WithIntroCache(enabled bool) Option {

	return client.WithIntroCache(enabled)

}
