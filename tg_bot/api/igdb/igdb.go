package igdb

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"tg_game_wishlist/api"
	"tg_game_wishlist/lib/e"
	"tg_game_wishlist/storage"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Finder struct {
	host          string
	clientId      string
	authorization string
	client        http.Client
}

const (
	gamesMethod    = "v4/games"
	gamesListParam = "search \"?\"; fields id,name,first_release_date; where version_parent = null & game_type = 0; limit 50;"
	gameParam      = "fields id,name,url,release_dates.date,release_dates.platform.abbreviation; where id = ?;"
)

func New(host, clientId, tokenType, token string) *Finder {
	return &Finder{
		host:          host,
		clientId:      clientId,
		authorization: newAuthorization(tokenType, token),
		client:        http.Client{},
	}
}

var authCaser = cases.Title(language.Und)

func newAuthorization(tokenType, token string) string {
	return authCaser.String(tokenType) + " " + token
}

func (f *Finder) Find(ctx context.Context, name string) (res []api.SearchResult, err error) {
	defer func() { err = e.WrapIfNil("can't find game list", err) }()

	reqBody := strings.ReplaceAll(gamesListParam, "?", name)
	log.Print(reqBody)

	data, err := f.doRequest(ctx, gamesMethod, nil, reqBody)
	if err != nil {
		return nil, err
	}

	var response SearchResponse

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	res = make([]api.SearchResult, 0, len(response))

	for _, game := range response {
		res = append(res, searchResult(game))
	}

	return res, nil
}

func searchResult(game Game) api.SearchResult {
	var res api.SearchResult
	res.Id = game.Id
	res.Name = game.Name
	res.FirstReleaseDate = game.FirstReleaseDate.Time

	return res
}

func (f *Finder) FindGameById(ctx context.Context, gameId int) (res *api.Game, err error) {
	defer func() { err = e.WrapIfNil("can't find one game data", err) }()

	reqBody := strings.ReplaceAll(gameParam, "?", strconv.Itoa(gameId))
	log.Print(reqBody)

	data, err := f.doRequest(ctx, gamesMethod, nil, reqBody)
	if err != nil {
		return nil, err
	}

	var response []Game

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return game(response[0]), nil
}

func game(response Game) *api.Game {
	res := &api.Game{
		Id:           response.Id,
		Name:         response.Name,
		URL:          response.URL,
		ReleaseDates: make([]api.PlatformDate, 0, len(response.ReleaseDates)),
		Source:       storage.Igdb,
	}

	for _, rDate := range response.ReleaseDates {
		res.ReleaseDates = append(res.ReleaseDates, releaseDate(rDate))
	}

	return res
}

func releaseDate(date ReleaseDate) api.PlatformDate {
	return api.PlatformDate{
		Platform: platform(date),
		Date:     date.Date.Time,
	}
}

func platform(date ReleaseDate) api.Platform {
	return api.Platform{
		Id:   date.Platform.Id,
		Name: date.Platform.Abbreviation,
	}
}

func (f *Finder) doRequest(ctx context.Context, method string, q url.Values, reqBody string) (data []byte, err error) {
	defer func() { err = e.WrapIfNil("can't do request", err) }()

	u := url.URL{
		Scheme: "https",
		Host:   f.host,
		Path:   method,
	}

	log.Print(u.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Client-ID", f.clientId)
	req.Header.Set("Authorization", f.authorization)

	req.URL.RawQuery = q.Encode()

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	log.Print(string(body))
	if err != nil {
		return nil, err
	}

	return body, nil
}
