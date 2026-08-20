package catalog

import (
	"sort"
	"strings"
)

// FindByID returns the channel with the given id, if present.
func (catalog *Catalog) FindByID(id string) (Channel, bool) {

	for _, channel := range catalog.Channels {

		if channel.ID == id {

			return channel, true

		}

	}

	return Channel{}, false

}

// FindByExactName returns the channel whose name matches query exactly, case-insensitively.
func (catalog *Catalog) FindByExactName(name string) (Channel, bool) {

	name = strings.ToLower(strings.TrimSpace(name))

	if name == "" {

		return Channel{}, false

	}

	for _, channel := range catalog.Channels {

		if strings.ToLower(channel.Name) == name {

			return channel, true

		}

	}

	for _, channel := range catalog.Channels {

		for _, alt := range channel.AltNames {

			if strings.ToLower(strings.TrimSpace(alt)) == name {

				return channel, true

			}

		}

	}

	return Channel{}, false

}

// Search returns channels whose name contains query, case-insensitively.
func (catalog *Catalog) Search(query string, limit int) []Channel {

	query = strings.ToLower(strings.TrimSpace(query))

	if query == "" {

		return nil

	}

	var matches []Channel

	for _, channel := range catalog.Channels {

		if channelMatchesQuery(channel, query) {

			matches = append(matches, channel)

		}

	}

	sort.Slice(matches, func(i, j int) bool {

		if matches[i].Enriched != matches[j].Enriched {

			return matches[i].Enriched

		}

		return strings.Compare(matches[i].Name, matches[j].Name) < 0

	})

	if limit > 0 && len(matches) > limit {

		matches = matches[:limit]

	}

	return matches

}

// Sorted returns channels ranked alphabetically, enriched channels first.
func (catalog *Catalog) Sorted() []Channel {

	channels := append([]Channel(nil), catalog.Channels...)

	sort.Slice(channels, func(i, j int) bool {

		if channels[i].Enriched != channels[j].Enriched {

			return channels[i].Enriched

		}

		return strings.Compare(channels[i].Name, channels[j].Name) < 0

	})

	return channels

}

// Popular returns a ranked subset of well-known channels.
func (catalog *Catalog) Popular(limit int) []Channel {

	ranked := rankPopular(catalog.Channels)

	if limit > 0 && len(ranked) > limit {

		ranked = ranked[:limit]

	}

	return ranked

}

func channelMatchesQuery(channel Channel, query string) bool {

	if strings.Contains(strings.ToLower(channel.Name), query) ||
		strings.Contains(strings.ToLower(channel.Slug), query) ||
		strings.Contains(strings.ToLower(channel.Network), query) {

		return true

	}

	for _, alt := range channel.AltNames {

		if strings.Contains(strings.ToLower(alt), query) {

			return true

		}

	}

	return false

}

func slugify(name string) string {

	name = strings.ToLower(name)

	var b strings.Builder
	lastWasSep := false

	for _, r := range name {

		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {

			b.WriteRune(r)
			lastWasSep = false

		} else if !lastWasSep && b.Len() > 0 {

			b.WriteByte('-')
			lastWasSep = true

		}

	}

	return strings.TrimRight(b.String(), "-")

}
