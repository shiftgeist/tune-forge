package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/shiftgeist/tune-forge/service"
	"github.com/shiftgeist/tune-forge/spotify"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	mux := http.NewServeMux()

	serviceSpotify := spotify.NewSpotifyService(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))
	serviceSpotify.RegisterRoutes(mux)

	http.HandleFunc("/spotify/", handleGetMe(serviceSpotify))
	http.HandleFunc("/spotify/playlists", handleGetPlaylists(serviceSpotify))

	log.Println("Now listening to http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGetMe(s service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := s.Me()
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}

		fmt.Fprintf(w, "Welcome \"%s\"", res.DisplayName)
	}
}

func handleGetPlaylists(s service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := s.Playlists()
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}

		t, err := template.ParseFiles("templates/playlists.html")
		err = t.Execute(w, res)
	}
}
