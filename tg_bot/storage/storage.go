package storage

import (
	"context"
	"errors"
	"time"
)

type Storage interface {
	Add(ctx context.Context, w *Wishlist) error
	IsExists(ctx context.Context, w *Wishlist) (bool, error)
	GetUserByName(ctx context.Context, userName string) (*User, error)
	GetAll(ctx context.Context, u *User) ([]Wishlist, error)
	GetReleased(ctx context.Context, u *User) ([]Wishlist, error)
	GetUnreleased(ctx context.Context, u *User) ([]Wishlist, error)
	Remove(ctx context.Context, w *Wishlist) error
	GetToNotify(ctx context.Context) ([]Wishlist, error)
	Notify(ctx context.Context, w *Wishlist) error
}

var (
	ErrNoWishlist = errors.New("no wishlist")
	ErrNoUser     = errors.New("user doesn't exist")
)

type Wishlist struct {
	Id                  int
	User                *User
	Game                *Game
	ExpectedReleaseDate time.Time
	AddedAt             time.Time
	NotifiedAt          time.Time
}

type Source int

const (
	Steam = iota
	Igdb
	Rawg
	Manual
)

type Game struct {
	Id          int
	Name        string
	Source      Source
	ExternalURL string
}

type User struct {
	Id   int
	Name string
}
