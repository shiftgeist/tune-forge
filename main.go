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

type Client struct {
	conf     oauth2.Config
	tok      oauth2.Token
	verifier string
	http     http.Client
}

func (c *Client) login(w http.ResponseWriter, r *http.Request) {
	url := c.conf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(c.verifier))
	http.Redirect(w, r, url, 302)
}

func (c *Client) callback(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")

	tok, err := c.conf.Exchange(ctx, code, oauth2.VerifierOption(c.verifier))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	c.tok = *tok
	c.http = *c.conf.Client(ctx, &c.tok)

	fmt.Fprintln(w, "You can close this now")
}

func newClient() *Client {
	clientId := strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_SECRET"))

	conf := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  "http://127.0.0.1:8080/callback",
		Scopes:       []string{"playlist-read-private", "playlist-read-collaborative", "playlist-modify-private", "playlist-modify-public", "user-library-read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	}

	return &Client{conf: *conf, verifier: oauth2.GenerateVerifier()}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	c := newClient()

	http.HandleFunc("/", helloWorld(c))
	http.HandleFunc("/login", c.login)
	http.HandleFunc("/callback", c.callback)

	log.Println("Now listening to http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func helloWorld(c *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := c.http.Get("https://api.spotify.com/v1/me")
		if err != nil {
			log.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode > 299 {
			log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, "%s", body)
	}
}
