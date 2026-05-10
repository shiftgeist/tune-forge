package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Fetch interface {
	Get(url string) (*http.Response, error)
}

type Image struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	URL    string `json:"url"`
}

type User struct {
	DisplayName string  `json:"display_name"`
	Id          string  `json:"id"`
	Images      []Image `json:"images"`
}

type Playlist struct {
	Id     string     `json:"id"`
	Name   string     `json:"name"`
	Images []Image    `json:"images"`
	Owner  User       `json:"owner"`
	Tracks TracksLink `json:"tracks"`
}

type PlaylistsMeta struct {
	Next  string     `json:"next"`
	Items []Playlist `json:"items"`
}

type TracksLink struct {
	Href  string `json:"href"`
	Total int    `json:"total"`
}

func fetchPage[T any](fetcher func(url string) (*http.Response, error), url string) (T, error) {
	var zero T

	res, err := fetcher(url)
	if err != nil {
		return zero, fmt.Errorf("get %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return zero, fmt.Errorf("get %s: status %d", url, res.StatusCode)
	}

	var page T
	if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
		return zero, fmt.Errorf("decode playlists: %w", err)
	}
	return page, nil
}

func GetMe(fetch Fetch) (User, error) {
	res, err := fetch.Get("https://api.spotify.com/v1/me")
	if err != nil {
		return User{}, err
	}
	if res.StatusCode > 299 {
		return User{}, errors.New("Not ok")
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return User{}, err
	}

	var dat User
	if err := json.Unmarshal(body, &dat); err != nil {
		return User{}, err
	}

	return dat, nil
}

func GetPlaylists(fetch Fetch) ([]Playlist, error) {
	var items []Playlist

	for url := "https://api.spotify.com/v1/me/playlists"; url != ""; {
		page, err := fetchPage[PlaylistsMeta](fetch.Get, url)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		url = page.Next
	}

	return items, nil
}
