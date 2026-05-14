package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/shiftgeist/tune-forge/auth"
	"github.com/shiftgeist/tune-forge/service"
)

var _ service.Service = (*spotify)(nil)

type spotify struct {
	client    *auth.Client
	playlists []service.Playlist

	routes service.Routes
}

func NewSpotifyService(clientId string, clientSecret string) *spotify {
	c := auth.NewClient(&auth.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  "http://127.0.0.1:8080/callback",
		Scopes:       []string{"playlist-read-private", "playlist-read-collaborative", "playlist-modify-private", "playlist-modify-public", "user-library-read"},
		Endpoint: auth.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	})

	return &spotify{
		client:    c,
		playlists: []service.Playlist{},
		routes: service.Routes{
			Login:         "",
			OAuthCallback: "",
			Me:            "",
			Playlists:     "",
		}}
}

func (s *spotify) Routes() *service.Routes {
	return &s.routes
}

func (s *spotify) RegisterRoutes(mux *http.ServeMux) {
	s.routes.Login = "/spotify/login"
	mux.HandleFunc(s.routes.Login, s.client.HandleLogin)

	s.routes.OAuthCallback = "/spotify/callback"
	mux.HandleFunc(s.routes.OAuthCallback, s.client.HandleCallback)
}

func (s *spotify) Me() (service.User, error) {
	res, err := s.client.Get("https://api.spotify.com/v1/me")
	if err != nil {
		return service.User{}, err
	}
	if res.StatusCode > 299 {
		return service.User{}, errors.New("Not ok")
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return service.User{}, err
	}

	var dat JsonUser
	if err := json.Unmarshal(body, &dat); err != nil {
		return service.User{}, err
	}

	imageUrl := ""
	if len(dat.Images) > 0 {
		imageUrl = dat.Images[0].URL
	}

	return service.User{
		DisplayName: dat.DisplayName,
		Id:          dat.Id,
		AvatarUrl:   imageUrl,
	}, nil
}

func (s *spotify) Playlists() ([]service.Playlist, error) {
	var items []service.Playlist

	for url := "https://api.spotify.com/v1/me/playlists"; url != ""; {
		page, err := fetchPage[JsonPlaylistsMeta](s.client.Get, url)
		if err != nil {
			return nil, err
		}

		for _, item := range page.Items {
			imageUrl := ""
			if len(item.Images) > 0 {
				imageUrl = item.Images[0].URL
			}

			items = append(items, service.Playlist{
				Id:       item.Id,
				CoverUrl: imageUrl,
				Name:     item.Name,
			})
		}

		url = page.Next
	}

	s.playlists = items
	return items, nil
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

type JsonImage struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	URL    string `json:"url"`
}

type JsonUser struct {
	DisplayName string      `json:"display_name"`
	Id          string      `json:"id"`
	Images      []JsonImage `json:"images"`
}

type JsonPlaylist struct {
	Id     string         `json:"id"`
	Name   string         `json:"name"`
	Images []JsonImage    `json:"images"`
	Owner  JsonUser       `json:"owner"`
	Tracks JsonTracksLink `json:"tracks"`
}

type JsonPlaylistsMeta struct {
	Next  string         `json:"next"`
	Items []JsonPlaylist `json:"items"`
}

type JsonTracksLink struct {
	Href  string `json:"href"`
	Total int    `json:"total"`
}
