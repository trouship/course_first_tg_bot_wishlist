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

func newAuthorization(token_type, token string) string {
	return strings.ToTitle(token_type) + token
}
