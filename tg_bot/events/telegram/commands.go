package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
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

	return p.searchGameList(ctx, text, chatID)
}

func (p *Processor) doCallback(ctx context.Context, callbackId string, text string, chatID int, userName string) (err error) {
	defer func() { err = e.WrapIfNil("can't process callback", err) }()

	parts := strings.Split(text, ":")

	err = p.tg.AnswerCallBack(ctx, callbackId, "Игра выбрана", false)
	if err != nil {
		return err
	}

	switch parts[0] {
	case "select":
		gameId, err := strconv.Atoi(parts[1])
		if err != nil {
			return err
		}

		game, err := p.gameById(ctx, gameId)
		if err != nil {
			return err
		}

		var buttons [][]telegram.InlineKeyboardButton

		for _, releaseDate := range game.ReleaseDates {
			button := telegram.InlineKeyboardButton{
				Text:         fmt.Sprintf("%s 📅 %s", releaseDate.Platform, releaseDate.Date.Format("02.01.2006")),
				CallbackData: fmt.Sprintf("add:%d:%d", game.Id, releaseDate.PlatformId),
			}
			buttons = append(buttons, []telegram.InlineKeyboardButton{button})
		}

		return p.tg.SendMessageWithKeyboard(ctx, chatID, fmt.Sprintf("🖥️ %s:", game.Name), &telegram.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		})
	}

	return nil
}

func (p *Processor) gameById(ctx context.Context, gameId int) (game *api.Game, err error) {
	defer func() { err = e.WrapIfNil("can't get game by id", err) }()

	game, err = p.finder.FindGameById(ctx, gameId)
	if err != nil {
		return nil, err
	}

	return game, nil
}

func (p *Processor) searchGameList(ctx context.Context, text string, chatID int) (err error) {
	defer func() { err = e.WrapIfNil("can't search game", err) }()

	var res []api.SearchResult
	res, err = p.finder.Find(ctx, text)
	if err != nil {
		return err
	}

	var buttons [][]telegram.InlineKeyboardButton

	for _, game := range res {
		button := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("🎮 %s", game.Name),
			CallbackData: fmt.Sprintf("select:%d", game.Id),
		}
		buttons = append(buttons, []telegram.InlineKeyboardButton{button})
	}

	return p.tg.SendMessageWithKeyboard(ctx, chatID, "Выберите игру: ", &telegram.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	})
}
