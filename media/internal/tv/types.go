package tv

import "time"

// Country is the enriched country metadata for a channel, backfilled from iptv-org.
type Country struct {

	Code string
	Name string

}

// Channel is a single 24/7 live TV channel from the ntv.cx catalog.
type Channel struct {

	ID   string `json:"channel_id"`
	Name string `json:"channel_name"`
	Code string `json:"channel_code"`
	Slug string

	Logo      string `json:"channel_image"`
	PlayerURL string `json:"channel_url"`

	Server string `json:"server"`

	Country  Country
	Category string
	Enriched bool

}

// reliableServer is the only ntv.cx backend whose channel_url reliably
// resolves to a playable stream (verified directly): "dlhd"-backed channels
// route through dlhd.st, whose CDN is currently returning 503s, and
// "hesgoales"-backed channels have no logos and an inconsistent stream CDN.
// Channels from other backends are dropped at fetch time rather than shown
// as entries that fail when a user tries to actually play them.
const reliableServer = "cdnlive"

// ChannelCatalog is the full set of channels fetched from ntv.cx.
type ChannelCatalog struct {

	Channels  []Channel
	FetchedAt time.Time

}

// getChannelsResponse mirrors the raw ntv.cx /api/get-channels payload.
type getChannelsResponse struct {

	Success  bool      `json:"success"`
	Channels []Channel `json:"channels"`

}
