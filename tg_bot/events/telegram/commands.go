package telegram

import (
	"bytes"
	"context"
	"log"
	"tg_game_wishlist/api"
	"tg_game_wishlist/lib/e"
)

const (
	SearchCmd = "/search"
)

func (p *Processor) doCmd(ctx context.Context, text string, chatID int, userName string) error {
	//text = strings.TrimSpace(text)

	log.Printf("got new commands '%s' from '%s'", text, userName)

	return p.searchGame(ctx, text, chatID, userName)
}

func (p *Processor) searchGame(ctx context.Context, text string, chatID int, userName string) (err error) {
	defer func() { err = e.WrapIfNil("can't search game", err) }()

	var res []api.SearchResult
	res, err = p.finder.Find(ctx, text)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer

	for _, game := range res {
		buffer.WriteString(game.Name + "\n")
	}

	return p.tg.SendMessage(ctx, chatID, buffer.String())
}
