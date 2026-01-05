package storage

import (
	"context"
	"time"
)

type Storage interface {
	Add(ctx context.Context, w *Wishlist) error
	GetUserByName(ctx context.Context, userName string) (User, error)
	GetAll(ctx context.Context, u *User) ([]Game, error)
	GetReleased(ctx context.Context, u *User) ([]Game, error)
	GetUnreleased(ctx context.Context, u *User) ([]Game, error)
	Remove(ctx context.Context, w *Wishlist) error
	GetToNotify(ctx context.Context) ([]Wishlist, error)
	Notify(ctx context.Context, w *Wishlist) error
}

type Wishlist struct {
	Id         int
	User       User
	Game       Game
	AddedAt    time.Time
	NotifiedAt time.Time
}

type Source int

const (
	Steam = iota
	igdb
	rawg
	manual
)

type Game struct {
	Id          int
	Name        string
	ReleaseDate time.Time
	Source      Source
	ExternalURL string
}

type User struct {
	Id   int
	Name string
}
