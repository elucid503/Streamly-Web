package showbox

// MediaType is the kind of catalogue entry Showbox search understands.
type MediaType string

const (

	MediaAll MediaType = "all"
	MediaMovie MediaType = "movie"
	MediaTV MediaType = "tv"

)

// BoxType is Showbox's discriminator between a movie and a series.
type BoxType int

const (

	BoxMovie BoxType = 1
	BoxSeries BoxType = 2

)

// SearchResult is a single hit from a Showbox search or autocomplete query.
type SearchResult struct {

	ID int `json:"id"`
	BoxType BoxType `json:"box_type"`

	Title string `json:"title"`

	Year int `json:"year,omitempty"`
	Poster string `json:"poster,omitempty"`
	Description string `json:"description,omitempty"`

	IMDBRating string `json:"imdb_rating,omitempty"`

}

// TopList is a curated Showbox ranking category.
type TopList struct {

	ID string `json:"id"`
	DisplayName string `json:"display_name"`

}
