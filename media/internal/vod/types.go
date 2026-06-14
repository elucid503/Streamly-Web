package vod

// MediaFile is a playable file inside a Febbox share.
type MediaFile struct {

	ID int

	Name string

	Season int
	Episode int

	shareKey string

}

func (f MediaFile) ShareKey() string { return f.shareKey }

// EpisodeInfo is display metadata for one episode.
type EpisodeInfo struct {

	Title string
	Description string

	Poster string

}
