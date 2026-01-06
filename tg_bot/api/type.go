package api

import (
	"context"
	"time"
)

type Finder interface {
	Find(ctx context.Context, name string) ([]SearchResult, error)
}

type SearchResult struct {
	Name         string
	URL          string
	ReleaseDates []PlatformDate
}

type PlatformDate struct {
	Platform string
	Date     time.Time
}
