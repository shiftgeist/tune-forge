package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/shiftgeist/tune-forge/auth"
	"github.com/shiftgeist/tune-forge/spotify"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	c := auth.NewClient(&auth.Config{
		ClientID:     strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_SECRET")),
		RedirectURL:  "http://127.0.0.1:8080/callback",
		Scopes:       []string{"playlist-read-private", "playlist-read-collaborative", "playlist-modify-private", "playlist-modify-public", "user-library-read"},
		Endpoint: auth.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	})
	http.HandleFunc("/spotify/login", c.HandleLogin)
	http.HandleFunc("/spotify/callback", c.HandleCallback)

	http.HandleFunc("/", handleGetMe(c))

	log.Println("Now listening to http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGetMe(c *auth.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := spotify.GetMe(c.Http)
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}

		fmt.Fprintf(w, "Welcome \"%s\"", res.DisplayName)
	}
}
