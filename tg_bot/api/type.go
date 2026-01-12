package api

import (
	"context"
	"tg_game_wishlist/storage"
	"time"
)

type Finder interface {
	Find(ctx context.Context, name string) ([]SearchResult, error)
	FindGameById(ctx context.Context, gameId int) (*Game, error)
}

type SearchResult struct {
	Id               int
	Name             string
	FirstReleaseDate time.Time
}

type Game struct {
	Id           int
	Name         string
	URL          string
	ReleaseDates []PlatformDate
	Source       storage.Source
}

type PlatformDate struct {
	Platform Platform
	Date     time.Time
}

type Platform struct {
	Id   int
	Name string
}
