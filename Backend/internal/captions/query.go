package captions

// Query identifies a title for subtitle lookup.
type Query struct {

	IMDBId string
	TMDBId int

	VideoName string

	Season int
	Episode int

}
