package source

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

var daddyConfigRE = regexp.MustCompile(`window\._econfig\s*=\s*["']([A-Za-z0-9+/=]+)["']`)

func extractDaddyConfigURL(body string) string {

	match := daddyConfigRE.FindStringSubmatch(body)

	if len(match) != 2 {

		return ""

	}

	encoded, err := base64.StdEncoding.DecodeString(match[1])

	if err != nil || len(encoded) % 4 != 0 || len(encoded) < 32 {

		return ""

	}

	// The player shuffles four padded base64 chunks and inserts one decoy byte in each.
	var chunks [4]string
	width := len(encoded) / 4

	for i, destination := range []int{2, 0, 3, 1} {

		part := string(encoded[i*width : (i+1)*width])
		decoded, err := base64.StdEncoding.DecodeString(part[:3] + part[4:])

		if err != nil {

			return ""

		}

		chunks[destination] = string(decoded)

	}

	data, err := base64.StdEncoding.DecodeString(strings.Join(chunks[:], ""))

	if err != nil {

		return ""

	}

	var config struct {

		StreamURL string `json:"stream_url"`
		DirectURL string `json:"stream_url_nop2p"`

	}

	if json.Unmarshal(data, &config) != nil {

		return ""

	}

	for _, raw := range []string{config.DirectURL, config.StreamURL} {

		u, err := url.Parse(strings.TrimSpace(raw))

		if err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && looksLikeHLS(u.Path) {

			return u.String()

		}

	}

	return ""

}
