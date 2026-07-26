package source

import "strings"

// PublicProvider is an anonymized source option exposed to API clients.
// Keys and labels must not reveal upstream brand or domain names.
type PublicProvider struct {

	// Key is the opaque token clients send back (e.g. "auto", "s1").
	Key string `json:"key"`

	// Label is a neutral display name (e.g. "Automatic", "Source 1").
	Label string `json:"label"`

	// Description is a short neutral hint for the UI.
	Description string `json:"description,omitempty"`

}

// publicMap maps internal provider names → public keys (stable).
// Never expose internal names over the wire.
var publicMap = []struct {
	Internal string
	Key string
	Label string
	Description string
}{

	{"daddylive", "s1", "Source 1", "Primary live feed"},
	{"ntv", "s2", "Source 2", "Alternate live feed"},
	{"pluto", "s3", "Source 3", "Free ad-supported feed"},
	{"iptvorg", "s4", "Source 4", "Open directory feed"},

}

// PublicKey returns the anonymized key for an internal provider name.
func PublicKey(internal string) string {

	internal = strings.ToLower(strings.TrimSpace(internal))

	for _, m := range publicMap {

		if m.Internal == internal {

			return m.Key

		}

	}

	return ""

}

// InternalName maps a public key back to the internal provider name.
func InternalName(publicKey string) string {

	publicKey = normalizePublicKey(publicKey)

	if publicKey == "" || publicKey == "auto" {

		return ""

	}

	for _, m := range publicMap {

		if m.Key == publicKey {

			return m.Internal

		}

	}

	return ""

}

// PublicLabel returns the display label for a public key.
func PublicLabel(publicKey string) string {

	publicKey = normalizePublicKey(publicKey)

	if publicKey == "" || publicKey == "auto" {

		return "Automatic"

	}

	for _, m := range publicMap {

		if m.Key == publicKey {

			return m.Label

		}

	}

	return "Source"

}

// PublicProviderList builds the client-facing provider list for a resolver.
// Only providers that are actually registered are included (plus Automatic).
func PublicProviderList(r *Resolver) []PublicProvider {

	out := []PublicProvider{

		{
			Key: "auto",
			Label: "Automatic",
			Description: "Try sources in preferred order",
		},

	}

	if r == nil {

		return out

	}

	registered := map[string]bool{}

	for _, name := range r.Providers() {

		registered[name] = true

	}

	for _, m := range publicMap {

		if !registered[m.Internal] {

			continue

		}

		out = append(out, PublicProvider{

			Key: m.Key,
			Label: m.Label,
			Description: m.Description,

		})

	}

	return out

}

func normalizePublicKey(key string) string {

	return strings.ToLower(strings.TrimSpace(key))

}
