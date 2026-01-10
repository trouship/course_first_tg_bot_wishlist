package api

import (
	"context"
	"time"
)

type Finder interface {
	Find(ctx context.Context, name string) ([]SearchResult, error)
	FindGameById(ctx context.Context, gameId int) (*Game, error)
}

type SearchResult struct {
	Id   int
	Name string
}

type Game struct {
	Id           int
	Name         string
	URL          string
	ReleaseDates []PlatformDate
}

type PlatformDate struct {
	PlatformId int
	Platform   string
	Date       time.Time
}
