package catalog

// Branding providers use that iptv-org omits from alt_names
// (e.g. DaddyLive "CW PIX 11 USA" → WPIXHD.us).
var channelExtras = map[string]struct {

	Name string
	AltNames []string

}{

	"WPIXHD.us": {

		Name: "PIX 11",
		AltNames: []string{"PIX11", "CW PIX 11", "WPIX"},

	},

}

func hasChannelExtras(id string) bool {

	_, ok := channelExtras[id]

	return ok

}

func applyChannelExtras(ch *Channel) {

	if ch == nil {

		return

	}

	extra, ok := channelExtras[ch.ID]

	if !ok {

		return

	}

	if extra.Name != "" && extra.Name != ch.Name {

		ch.AltNames = append(ch.AltNames, ch.Name)
		ch.Name = extra.Name
		ch.Slug = slugify(extra.Name)

	}

	ch.AltNames = mergeAltNames(ch.AltNames, extra.AltNames, ch.Name)

}
