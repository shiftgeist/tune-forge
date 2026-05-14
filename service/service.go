package service

type Routes struct {
	Login         string
	Me            string
	Playlists     string
	OAuthCallback string
}

type User struct {
	DisplayName string
	Id          string
	AvatarUrl   string
}

type Playlist struct {
	Id       string
	Name     string
	CoverUrl string
}

type Service interface {
	Routes() *Routes
	Me() (User, error)
	Playlists() ([]Playlist, error)
}
