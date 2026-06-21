package tv

import (
	"encoding/json"
	"strings"
)

// Country holds channel country metadata from tv-channels.json.
type Country struct {

	Code string `json:"code"`
	Name string `json:"name"`
	Flag string `json:"flag"`

}

// UnmarshalJSON accepts both the current object shape and the older/string shape returned by some catalog providers.
func (c *Country) UnmarshalJSON(data []byte) error {

	var country struct {

		Code string `json:"code"`
		Name string `json:"name"`
		Flag string `json:"flag"`

	}

	if err := json.Unmarshal(data, &country); err == nil {

		c.Code = strings.ToLower(strings.TrimSpace(country.Code))
		c.Name = strings.TrimSpace(country.Name)
		c.Flag = strings.TrimSpace(country.Flag)

		if c.Name == "" {

			c.Name = strings.ToUpper(c.Code)

		}

		return nil

	}

	var name string

	if err := json.Unmarshal(data, &name); err != nil {

		return err

	}

	name = strings.TrimSpace(name)
	code := countryCode(name)

	c.Code = code
	c.Name = name
	c.Flag = ""

	return nil

}

func countryCode(value string) string {

	normalized := strings.ToLower(strings.TrimSpace(value))

	switch normalized {

		case "united states", "usa", "u.s.", "u.s.a.":

			return "us"

		case "united kingdom", "uk", "great britain":

			return "gb"

	}

	if len(normalized) == 2 {

		return normalized

	}

	return ""

}

// Channel is a single Live TV entry from the catalog.
type Channel struct {

	ID string `json:"id"`
	DaddyID string `json:"daddyId"`

	Name string `json:"name"`
	Slug string `json:"slug"`

	Logo string `json:"logo"`

	Country Country `json:"country"`
	Category string `json:"category"`

	Status string `json:"status"`
	Source string `json:"source"`

}

// UnmarshalJSON tolerates provider schema drift: newer catalogs name the logo field "image", omit "id", or send "id" as a number, so we accept both logo keys, coerce numeric ids to strings, and synthesize an id from daddyId when absent.
func (c *Channel) UnmarshalJSON(data []byte) error {

	type channelAlias Channel

	var raw struct {

		channelAlias

		// RawID shadows channelAlias.ID so we can accept string or number.
		RawID json.RawMessage `json:"id"`

		Image string `json:"image"`

	}

	if err := json.Unmarshal(data, &raw); err != nil {

		return err

	}

	*c = Channel(raw.channelAlias)

	if len(raw.RawID) > 0 {

		var s string

		if err := json.Unmarshal(raw.RawID, &s); err == nil {

			c.ID = s

		} else {

			// numeric id — use the raw digits as the string value
			c.ID = strings.TrimSpace(string(raw.RawID))

		}

	}

	if c.Logo == "" {

		c.Logo = strings.TrimSpace(raw.Image)

	}

	if c.ID == "" {

		c.ID = c.DaddyID

	}

	return nil

}

// ChannelCatalog is the top-level response from tv-channels.json.
type ChannelCatalog struct {

	Generated string `json:"generated"`

	Total int `json:"total"`
	Source string `json:"source"`

	StreamAPI string `json:"streamApi"`

	Channels []Channel `json:"channels"`

}

// ResolveResult is the legacy dami-tv.pro resolve payload.
type ResolveResult struct {

	Success bool   `json:"success"`
	Stream  string `json:"stream"`

	Error string `json:"error"`

}

// TV247ResolveResult is returned by the tv247 resolve-dlstream API.
type TV247ResolveResult struct {

	ChannelID string `json:"channelId"`

	ProxyPlaylistURL string `json:"proxyPlaylistUrl"`
	ProxyPlayerURL string `json:"proxyPlayerUrl"`

	Error string `json:"error"`

}

// ResolvedStream is a live TV playlist URL and the Referer it expects.
type ResolvedStream struct {

	URL string
	Referer string

}

// StreamInfo pairs a channel with its resolved HLS playlist URL.
type StreamInfo struct {

	Channel Channel
	HLSURL string

}
