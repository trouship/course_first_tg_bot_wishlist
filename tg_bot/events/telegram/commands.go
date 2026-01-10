package telegram

import (
	"context"
	"fmt"
	"log"
	"tg_game_wishlist/api"
	"tg_game_wishlist/clients/telegram"
	"tg_game_wishlist/lib/e"
)

const (
	SearchCmd = "/search"
)

func (p *Processor) doCmd(ctx context.Context, text string, chatID int, userName string) error {
	//text = strings.TrimSpace(text)

	log.Printf("got new command '%s' from '%s'", text, userName)

	return p.searchGame(ctx, text, chatID)
}

func (p *Processor) doCallback(ctx context.Context, callbackId string, text string, chatID int, userName string) error {
	p.tg.AnswerCallBack(ctx, callbackId, "Игра выбрана", false)

	return nil
}

func (p *Processor) searchGame(ctx context.Context, text string, chatID int) (err error) {
	defer func() { err = e.WrapIfNil("can't search game", err) }()

	var res []api.SearchResult
	res, err = p.finder.Find(ctx, text)
	if err != nil {
		return err
	}

	var buttons [][]telegram.InlineKeyboardButton

	for i, game := range res {
		button := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("🎮 %s", game.Name),
			CallbackData: fmt.Sprintf("add:%d", i),
		}
		buttons = append(buttons, []telegram.InlineKeyboardButton{button})
	}

	return p.tg.SendMessageWithKeyboard(ctx, chatID, "Выберите игру: ", &telegram.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	})
}
