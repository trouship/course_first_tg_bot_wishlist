package igdb

import (
	"net/http"
	"strings"
)

type Finder struct {
	host          string
	clientId      string
	authorization string
	client        http.Client
}

func newAuthorization(tokenType, token string) string {
	return strings.ToTitle(tokenType) + token
}
