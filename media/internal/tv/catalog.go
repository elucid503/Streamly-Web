package tv

import (
	"sort"
	"strings"
)

// FindByID returns the channel with the given id, if present.
func (catalog *ChannelCatalog) FindByID(id string) (Channel, bool) {

	for _, channel := range catalog.Channels {

		if channel.ID == id {

			return channel, true

		}

	}

	return Channel{}, false

}

// FindByExactName returns the channel whose name matches query exactly, case-insensitively.
func (catalog *ChannelCatalog) FindByExactName(name string) (Channel, bool) {

	name = strings.ToLower(strings.TrimSpace(name))

	if name == "" {

		return Channel{}, false

	}

	for _, channel := range catalog.Channels {

		if strings.ToLower(channel.Name) == name {

			return channel, true

		}

	}

	return Channel{}, false

}

// Search returns channels whose name contains query, case-insensitively.
func (catalog *ChannelCatalog) Search(query string, limit int) []Channel {

	query = strings.ToLower(strings.TrimSpace(query))

	if query == "" {

		return nil

	}

	var matches []Channel

	for _, channel := range catalog.Channels {

		if strings.Contains(strings.ToLower(channel.Name), query) {

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

// Sorted returns channels ranked alphabetically, channels with a known icon first.
func (catalog *ChannelCatalog) Sorted() []Channel {

	channels := append([]Channel(nil), catalog.Channels...)

	sort.Slice(channels, func(i, j int) bool {

		if channels[i].Enriched != channels[j].Enriched {

			return channels[i].Enriched

		}

		return strings.Compare(channels[i].Name, channels[j].Name) < 0

	})

	return channels

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
