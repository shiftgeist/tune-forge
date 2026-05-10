package spotify

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type Fetch interface {
	Get(url string) (*http.Response, error)
}

type SpotifyUser struct {
	DisplayName string `json:"display_name"`
	Id          string `json:"id"`
	Images      []struct {
		Height int    `json:"height"`
		Width  int    `json:"width"`
		URL    string `json:"url"`
	} `json:"images"`
}

func GetMe(fetch Fetch) (SpotifyUser, error) {
	res, err := fetch.Get("https://api.spotify.com/v1/me")
	if err != nil {
		return SpotifyUser{}, err
	}
	if res.StatusCode > 299 {
		return SpotifyUser{}, errors.New("Not ok")
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return SpotifyUser{}, err
	}

	var dat SpotifyUser
	if err := json.Unmarshal(body, &dat); err != nil {
		return SpotifyUser{}, err
	}

	return dat, nil
}
