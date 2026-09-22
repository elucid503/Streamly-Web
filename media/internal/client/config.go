package client

import (
	"os"
)

// Option configures a Client.
type Option func(*config)

type config struct {

	childMode string

	febboxCookie string
	introDBKey string
	tmdbAPIKey string

	cacheIntro bool

}

// WithChildMode sets the Showbox child-mode flag.
func WithChildMode(mode string) Option {

	return func(c *config) { c.childMode = mode }

}

// WithFebboxCookie sets the Febbox `ui` auth cookie required for quality links.
func WithFebboxCookie(cookie string) Option {

	return func(c *config) { c.febboxCookie = cookie }

}

// WithIntroDBKey sets an optional TheIntroDB API key.
func WithIntroDBKey(key string) Option {

	return func(c *config) { c.introDBKey = key }

}

// WithTMDBAPIKey sets the TMDB v3 API key for episode and title metadata.
func WithTMDBAPIKey(key string) Option {

	return func(c *config) { c.tmdbAPIKey = key }

}

// WithIntroCache enables a 6-hour in-memory cache for TheIntroDB lookups.
func WithIntroCache(enabled bool) Option {

	return func(c *config) { c.cacheIntro = enabled }

}

func applyDefaults(c *config) {

	if c.childMode == "" {

		c.childMode = os.Getenv("CHILD_MODE")

	}

	if c.childMode == "" {

		c.childMode = "0"

	}

	if c.febboxCookie == "" {

		c.febboxCookie = os.Getenv("FEBBOX_UI_COOKIE")

	}

	if c.introDBKey == "" {

		c.introDBKey = os.Getenv("INTRODB_API_KEY")

	}

	if c.tmdbAPIKey == "" {

		c.tmdbAPIKey = os.Getenv("TMDB_API_KEY")

	}

}
