package catalog

import "time"

// Country is ISO country metadata for a channel.
type Country struct {

	Code string
	Name string

}

// Channel is metadata-only identity for a live TV station.
// It has no stream URLs or provider references.
type Channel struct {

	ID string
	Name string
	Slug string

	// Code is a short country/region code (e.g. "us").
	Code string

	Logo string

	Country Country

	// Category is the primary display category.
	Category string

	// Categories is the full set of tags from the metadata source.
	Categories []string

	Network string
	Owners []string
	Website string
	AltNames []string

	// Enriched is true when the channel was built from a full metadata source
	// (logos, categories, owners) rather than a bare name seed.
	Enriched bool

}

// Catalog is an immutable snapshot of the channel index.
type Catalog struct {

	Channels []Channel
	FetchedAt time.Time

}
