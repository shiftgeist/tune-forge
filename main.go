package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

type Client struct {
	conf        oauth2.Config
	verifier    string
	http        http.Client
	storagePath string
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

	c.saveSession(*tok)
	c.http = *oauth2.NewClient(ctx, newTokenSource(c, c.conf.TokenSource(ctx, tok)))
	fmt.Fprintln(w, "You can close this now")
}

func (c *Client) saveSession(session oauth2.Token) error {
	j, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return os.WriteFile(c.storagePath, j, 0644)
}

func (c *Client) getSession() (*oauth2.Token, error) {
	byt, err := os.ReadFile(c.storagePath)
	if err != nil {
		return nil, err
	}

	var dat oauth2.Token
	if err := json.Unmarshal(byt, &dat); err != nil {
		return nil, err
	}

	return &dat, nil
}

func newClient() *Client {
	ctx := context.Background()

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

	c := &Client{conf: *conf, verifier: oauth2.GenerateVerifier(), storagePath: filepath.Join(os.TempDir(), "tune-forge-session.json")}

	persistantTok, err := c.getSession()
	if err != nil {
		return c
	}

	c.http = *oauth2.NewClient(ctx, newTokenSource(c, conf.TokenSource(ctx, persistantTok)))
	return c
}

type myTokenSource struct {
	client *Client
	inner  oauth2.TokenSource
}

func newTokenSource(client *Client, inner oauth2.TokenSource) myTokenSource {
	return myTokenSource{client: client, inner: inner}
}

func (s myTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.inner.Token()
	if err == nil {
		s.client.saveSession(*tok)
	}
	return tok, err
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
			http.Error(w, err.Error(), 502)
			return
		}
		if res.StatusCode > 299 {
			http.Error(w, "Response failed", res.StatusCode)
			return
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}

		fmt.Fprintf(w, "%s", body)
	}
}
