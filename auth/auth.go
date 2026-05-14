package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

type Config = oauth2.Config
type Endpoint = oauth2.Endpoint

type Client struct {
	conf        oauth2.Config
	verifier    string
	storagePath string
	http        *http.Client
}

var ErrNotAuthenticated = errors.New("not authenticated")

func (c *Client) HandleLogin(w http.ResponseWriter, r *http.Request) {
	url := c.conf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(c.verifier))
	http.Redirect(w, r, url, 302)
}

func (c *Client) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")

	tok, err := c.conf.Exchange(ctx, code, oauth2.VerifierOption(c.verifier))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	c.saveSession(*tok)
	c.http = oauth2.NewClient(ctx, newTokenSource(c, c.conf.TokenSource(ctx, tok)))
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

func (c *Client) Get(url string) (*http.Response, error) {
	if c.http == nil {
		return nil, ErrNotAuthenticated
	}
	return c.http.Get(url)
}

func (c *Client) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	if c.http == nil {
		return nil, ErrNotAuthenticated
	}
	return c.http.Post(url, contentType, body)
}

func NewClient(conf *oauth2.Config) *Client {
	ctx := context.Background()

	c := &Client{conf: *conf, verifier: oauth2.GenerateVerifier(), storagePath: filepath.Join(os.TempDir(), "tune-forge-session.json")}

	persistantTok, err := c.getSession()
	if err != nil {
		return c
	}

	c.http = oauth2.NewClient(ctx, newTokenSource(c, conf.TokenSource(ctx, persistantTok)))
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
