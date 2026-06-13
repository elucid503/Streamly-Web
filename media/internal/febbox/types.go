package febbox

// File is an entry in a Febbox shared folder; either a file or a sub-folder.
type File struct {

	FID int `json:"fid"`
	FileName string `json:"file_name"`

	IsDir int `json:"is_dir"` // 1 for directory, 0 for playable file.

}

// Quality is one downloadable rendition of a Febbox video.
type Quality struct {

	URL string `json:"url"`
	Quality string `json:"quality"`

	Name string `json:"name"`

	Speed string `json:"speed"`
	Size string `json:"size"`

}
