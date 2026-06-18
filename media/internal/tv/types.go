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
