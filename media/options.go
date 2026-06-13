package mediakit

import (
	"os"

	"mediakit/internal/febbox"
	"mediakit/internal/introdb"
	"mediakit/internal/showbox"
	"mediakit/internal/tv"
)

// Option configures a Client.
type Option func(*config)

type config struct {

	childMode string

	febboxCookie string
	introDBKey string

	tvBaseURL string

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

// WithTVBaseURL overrides the live TV catalog origin.
func WithTVBaseURL(baseURL string) Option {

	return func(c *config) { c.tvBaseURL = baseURL }

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

	if c.tvBaseURL == "" {

		c.tvBaseURL = os.Getenv("TV_BASE_URL")

	}

}

func (c *config) showboxOptions() showbox.Options {

	return showbox.Options{ChildMode: c.childMode}

}

func (c *config) febboxOptions() febbox.Options {

	return febbox.Options{Cookie: c.febboxCookie}

}

func (c *config) tvOptions() tv.Options {

	return tv.Options{BaseURL: c.tvBaseURL}

}

func (c *config) introDBOptions() introdb.Options {

	return introdb.Options{APIKey: c.introDBKey}

}
