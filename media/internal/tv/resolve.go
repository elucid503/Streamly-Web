package tv

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var (
	varDeclRE = regexp.MustCompile(`var\s+(\w+)\s*=\s*'([^']*)';`)
	chainRE   = regexp.MustCompile(`var\s+\w+\s*=\s*((?:\w+\(\w+\)\s*\+\s*)*\w+\(\w+\))\s*;`)
	callRE    = regexp.MustCompile(`\w+\((\w+)\)`)
)

// ResolveStream fetches a channel's player page and reconstructs the HLS
// playlist URL. The player page embeds the URL as a handful of base64
// literals assigned to randomly-named vars, concatenated in one expression
// (e.g. `var X=decode(a)+decode(b)+...;`) — this walks that structure instead
// of executing the page's JS.
func (c *Client) ResolveStream(playerURL string) (string, error) {

	playerURL = strings.TrimSpace(playerURL)

	if playerURL == "" {

		return "", fmt.Errorf("tv: player url is required")

	}

	response, err := c.get(playerURL)

	if err != nil {

		return "", fmt.Errorf("tv: fetch player page: %w", err)

	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {

		return "", fmt.Errorf("tv: read player page: %w", err)

	}

	if response.StatusCode != http.StatusOK {

		return "", fmt.Errorf("tv: fetch player page: status %d", response.StatusCode)

	}

	streamURL, err := extractStreamURL(string(body))

	if err != nil {

		return "", err

	}

	return streamURL, nil

}

func extractStreamURL(html string) (string, error) {

	literals := make(map[string]string)

	for _, match := range varDeclRE.FindAllStringSubmatch(html, -1) {

		literals[match[1]] = match[2]

	}

	if len(literals) == 0 {

		return "", fmt.Errorf("tv: no base64 literals found in player page")

	}

	var best []string

	for _, chain := range chainRE.FindAllStringSubmatch(html, -1) {

		names := make([]string, 0)
		ok := true

		for _, call := range callRE.FindAllStringSubmatch(chain[1], -1) {

			if _, exists := literals[call[1]]; !exists {

				ok = false
				break

			}

			names = append(names, call[1])

		}

		if ok && len(names) > len(best) {

			best = names

		}

	}

	if len(best) == 0 {

		return "", fmt.Errorf("tv: no stream url expression found in player page")

	}

	var builder strings.Builder

	for _, name := range best {

		decoded, err := decodeURLSafeBase64(literals[name])

		if err != nil {

			return "", fmt.Errorf("tv: decode stream url fragment: %w", err)

		}

		builder.Write(decoded)

	}

	streamURL := builder.String()

	if !strings.HasPrefix(streamURL, "http://") && !strings.HasPrefix(streamURL, "https://") {

		return "", fmt.Errorf("tv: decoded stream url looks invalid: %q", streamURL)

	}

	return streamURL, nil

}

func decodeURLSafeBase64(value string) ([]byte, error) {

	value = strings.ReplaceAll(value, "-", "+")
	value = strings.ReplaceAll(value, "_", "/")

	for len(value)%4 != 0 {

		value += "="

	}

	return base64.StdEncoding.DecodeString(value)

}
