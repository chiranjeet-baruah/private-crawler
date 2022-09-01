package types

// Images represents Images object in json response
type Images struct {
	Primary  ImageType   `json:"Primary,omitempty"`
	Variants []ImageType `json:"Variants,omitempty"`
}

// ImageType represents ImageType object in json response
type ImageType struct {
	Small  ImageSize `json:"Small,omitempty"`
	Medium ImageSize `json:"Medium,omitempty"`
	Large  ImageSize `json:"Large,omitempty"`
}

// ImageSize represents ImageSize object in json response
type ImageSize struct {
	URL    string `json:"URL,omitempty"`
	Height int    `json:"Height,omitempty"`
	Width  int    `json:"Width,omitempty"`
}

// GetImages returns images primary and variants
// first image in the array is always primary
func (i Images) GetImages() []string {
	res := []string{}
	primaryImg := i.Primary.Large.URL
	res = append(res, primaryImg)

	for _, img := range i.Variants {
		res = append(res, img.Large.URL)
	}
	return res
}
