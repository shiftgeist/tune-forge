package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	clientId := strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_SECRET"))

	conf := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  "http://127.0.0.1:8080/callback",
		Scopes:       []string{"user-library-read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	}

	verifier := oauth2.GenerateVerifier()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello world!\n")
	})
	http.HandleFunc("/login", login(conf, verifier))
	http.HandleFunc("/callback", makeCallbackHandler(conf, verifier))

	log.Println("Now listening to http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func login(conf *oauth2.Config, verifier string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
		http.Redirect(w, r, url, 302)
	}
}

func makeCallbackHandler(conf *oauth2.Config, verifier string) http.HandlerFunc {
	ctx := context.Background()

	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")

		fmt.Fprintf(w, "Code %q", code)

		tok, err := conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
		if err != nil {
			log.Fatal(err)
		}

		client := conf.Client(ctx, tok)
		client.Get("?") // wdid?

		// demo had all in main with fmt.Scan for code?
		// "Exchange will do the handshake to retrieve the initial access token. The HTTP Client returned by conf.Client will refresh the token as necessary." -> how does code refresh?
	}
}
