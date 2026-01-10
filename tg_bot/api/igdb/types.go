package igdb

import (
	"encoding/json"
	"tg_game_wishlist/lib/e"
	"time"
)

type SearchResponse []Game

type Game struct {
	Id           int           `json:"id"`
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	ReleaseDates []ReleaseDate `json:"release_dates"`
}

type ReleaseDate struct {
	Date     UnixTime `json:"date"`
	Platform Platform `json:"platform"`
}

type UnixTime struct {
	time.Time
}

func (ut *UnixTime) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return e.Wrap("can't unmarshal json unix time", err)
	}
	ut.Time = time.Unix(timestamp, 0)
	return nil
}

type Platform struct {
	Id           int    `json:"id"`
	Abbreviation string `json:"abbreviation"`
}
