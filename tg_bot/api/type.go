package api

import "time"

type Finder interface {
	Find(name string) ([]SearchResult, error)
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
