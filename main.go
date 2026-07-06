package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/shiftgeist/tune-forge/service"
	"github.com/shiftgeist/tune-forge/spotify"
)

//go:embed templates/*.html
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	mux := http.NewServeMux()

	serviceSpotify := spotify.NewSpotifyService(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))

	// TODO: Just register all routes in here no RegisterRoutes
	serviceSpotify.RegisterRoutes(mux)

	serviceSpotify.Routes().Me = "/spotify/"
	mux.HandleFunc(serviceSpotify.Routes().Me, serviceSpotify.RequireAuth(handleGetMe(serviceSpotify)))

	serviceSpotify.Routes().Playlists = "/spotify/playlists"
	mux.HandleFunc(serviceSpotify.Routes().Playlists, serviceSpotify.RequireAuth(handleGetPlaylists(serviceSpotify)))

	mux.HandleFunc("/", handleHome(serviceSpotify))

	log.Println("Now listening to http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleHome(s service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%+v", s.Routes())
	}
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
		auth_status := r.URL.Query().Get("auth")
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}

		data := struct {
			Status    string
			Playlists []service.Playlist
		}{
			Status:    "Auth: " + auth_status,
			Playlists: res,
		}

		render(w, "layout.html", data)
	}
}
