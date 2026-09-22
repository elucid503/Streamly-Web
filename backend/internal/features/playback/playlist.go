package playback

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	hlsURIAttr = regexp.MustCompile(`URI="([^"]+)"`)
	hlsAudioLangRE = regexp.MustCompile(`(?i)LANGUAGE="([^"]+)"`)
	hlsDefaultRE = regexp.MustCompile(`(?i)(DEFAULT=)(YES|NO)`)
	hlsAutoselectRE = regexp.MustCompile(`(?i)(AUTOSELECT=)(YES|NO)`)
	hlsSubsAttrRE = regexp.MustCompile(`(?i),?SUBTITLES="[^"]*"`)
	hlsGroupIDRE = regexp.MustCompile(`(?i)GROUP-ID="([^"]+)"`)
)

func (s *ProxyService) proxyMediaURL(base *url.URL, entry *ProxyEntry, baseProxyURL, raw string) (string, error) {

	trimmed := strings.TrimSpace(raw)

	if trimmed == "" || strings.HasPrefix(trimmed, "data:") {

		return trimmed, nil

	}

	resolved := resolveRelativeURL(base, trimmed)

	token, err := s.getOrCreateTokenWithHeaders(resolved, entry.Referer, entry.RequestHeaders)

	if err != nil {

		return "", err

	}

	return baseProxyURL + "/api/proxy/" + token, nil

}

func (s *ProxyService) RewritePlaylist(body []byte, entry *ProxyEntry, baseProxyURL string) []byte {

	text := strings.ReplaceAll(string(body), "\r\n", "\n")

	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")

	base, _ := url.Parse(entry.TargetURL)

	englishGroups := audioGroupsWithEnglish(lines)
	defaulted := make(map[string]bool, len(englishGroups))

	out := make([]string, 0, len(lines))

	for _, line := range lines {

		trimmed := strings.TrimSpace(line)

		if trimmed == "" {

			out = append(out, "")

			continue

		}

		if strings.HasPrefix(trimmed, "#") {

			if strings.Contains(trimmed, "EXT-X-MEDIA") && strings.Contains(trimmed, "TYPE=SUBTITLES") {

				continue

			}

			rewritten := line

			if strings.Contains(trimmed, "EXT-X-MEDIA") && strings.Contains(trimmed, "TYPE=AUDIO") {

				group := firstSubmatch(hlsGroupIDRE, trimmed)

				// Only re-point a group that actually has an English track. Forcing
				// DEFAULT=NO across a group with no English member leaves it with no
				// default at all, and AVFoundation rejects the whole playlist.
				if englishGroups[group] {

					want := !defaulted[group] && isEnglishAudioLang(firstSubmatch(hlsAudioLangRE, trimmed))

					rewritten = setAudioDefault(rewritten, want)

					if want {

						defaulted[group] = true

					}

				}

			}

			// The subtitle EXT-X-MEDIA lines are dropped above; AVFoundation
			// (Apple TV) rejects the whole playlist if a variant still points at
			// the group that no longer exists.
			if strings.Contains(trimmed, "EXT-X-STREAM-INF") {

				rewritten = hlsSubsAttrRE.ReplaceAllString(rewritten, "")

			}

			rewritten = hlsURIAttr.ReplaceAllStringFunc(rewritten, func(match string) string {

				parts := hlsURIAttr.FindStringSubmatch(match)

				if len(parts) < 2 {

					return match

				}

				proxyURL, err := s.proxyMediaURL(base, entry, baseProxyURL, parts[1])

				if err != nil {

					return match

				}

				return `URI="` + proxyURL + `"`

			})

			out = append(out, rewritten)

			continue

		}

		proxyURL, err := s.proxyMediaURL(base, entry, baseProxyURL, trimmed)

		if err != nil {

			out = append(out, line)

			continue

		}

		out = append(out, proxyURL)

	}

	return []byte(strings.Join(out, "\n"))

}

func resolveRelativeURL(base *url.URL, ref string) string {

	parsed, err := url.Parse(ref)

	if err != nil {

		return ref

	}

	return base.ResolveReference(parsed).String()

}

// audioGroupsWithEnglish reports the audio GROUP-IDs that contain at least one
// English rendition, so every other group keeps the source's own defaults.
func audioGroupsWithEnglish(lines []string) map[string]bool {

	groups := map[string]bool{}

	for _, line := range lines {

		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, "#EXT-X-MEDIA") || !strings.Contains(trimmed, "TYPE=AUDIO") {

			continue

		}

		group := firstSubmatch(hlsGroupIDRE, trimmed)

		if group == "" || !isEnglishAudioLang(firstSubmatch(hlsAudioLangRE, trimmed)) {

			continue

		}

		groups[group] = true

	}

	return groups

}

// setAudioDefault marks one English rendition per group as the default so
// players pick English on their own. Exactly one member of a group may say YES.
func setAudioDefault(line string, want bool) string {

	value := "NO"

	if want {

		value = "YES"

	}

	line = hlsDefaultRE.ReplaceAllString(line, "${1}"+value)
	line = hlsAutoselectRE.ReplaceAllString(line, "${1}"+value)

	return line

}

func firstSubmatch(re *regexp.Regexp, s string) string {

	match := re.FindStringSubmatch(s)

	if len(match) < 2 {

		return ""

	}

	return match[1]

}

func isEnglishAudioLang(lang string) bool {

	switch strings.ToLower(strings.TrimSpace(lang)) {

	case "en", "eng", "english":

		return true

	}

	return false

}
