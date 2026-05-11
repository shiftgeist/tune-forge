package service

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
	Me() (User, error)
	Playlists() ([]Playlist, error)
}
