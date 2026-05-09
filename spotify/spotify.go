package spotify

type User struct {
	DisplayName string `json:"display_name"`
	Id          string `json:"id"`
	Images      []struct {
		Height int    `json:"height"`
		Width  int    `json:"width"`
		URL    string `json:"url"`
	} `json:"images"`
}
