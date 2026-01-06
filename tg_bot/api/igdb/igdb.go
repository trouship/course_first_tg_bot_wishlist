package igdb

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"tg_game_wishlist/api"
	"tg_game_wishlist/lib/e"
)

type Finder struct {
	host          string
	clientId      string
	authorization string
	client        http.Client
}

const (
	gamesMethod = "games"
	gamesParam  = "; fields name,url,release_dates.date,release_dates.platform.abbreviation; where version_parent = null;"
)

func New(host, clientId, tokenType, token string) *Finder {
	return &Finder{
		host:          host,
		clientId:      clientId,
		authorization: newAuthorization(tokenType, token),
		client:        http.Client{},
	}
}

func newAuthorization(tokenType, token string) string {
	return strings.ToTitle(tokenType) + token
}

func (f *Finder) Find(ctx context.Context, name string) (res []api.SearchResult, err error) {
	defer func() { err = e.WrapIfNil("can't find game", err) }()

	reqBody := "search" + name + gamesParam

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
	res.URL = game.URL
	res.Name = game.Name
	res.ReleaseDates = make([]api.PlatformDate, 0, len(game.ReleaseDates))
	for _, rDate := range game.ReleaseDates {
		res.ReleaseDates = append(res.ReleaseDates, releaseDate(rDate))
	}

	return res
}

func releaseDate(date ReleaseDate) api.PlatformDate {
	return api.PlatformDate{
		Platform: date.Platform.Abbreviation,
		Date:     date.Date.Time,
	}
}

func (f *Finder) doRequest(ctx context.Context, method string, q url.Values, reqBody string) (data []byte, err error) {
	defer func() { err = e.WrapIfNil("can't do request", err) }()

	u := url.URL{
		Scheme: "https",
		Host:   f.host,
		Path:   method,
	}

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
	if err != nil {
		return nil, err
	}

	return body, nil
}
