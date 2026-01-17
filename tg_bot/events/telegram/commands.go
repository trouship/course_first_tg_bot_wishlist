package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"tg_game_wishlist/api"
	"tg_game_wishlist/clients/telegram"
	"tg_game_wishlist/lib/e"
	"tg_game_wishlist/storage"
)

const (
	HelpCmd   = "/help"
	StartCmd  = "/start"
	ListCmd   = "/list"
	RemoveCmd = "/remove"
)

const (
	SelectCallback = "select"
	AddCallback    = "add"
	RemoveCallback = "remove"
)

func (p *Processor) doCmd(ctx context.Context, text string, chatID int, userName string) error {
	//text = strings.TrimSpace(text)

	log.Printf("got new command '%s' from '%s'", text, userName)

	switch text {
	case HelpCmd:
		return p.sendHelp(ctx, chatID)
	case StartCmd:
		return p.sendHello(ctx, chatID)
	case ListCmd:
		return p.sendGameList(ctx, chatID, userName)
	case RemoveCmd:
		return p.sendRemoveList(ctx, chatID, userName)
	default:
		return p.searchGameList(ctx, text, chatID)
	}
}

func (p *Processor) sendRemoveList(ctx context.Context, chatId int, userName string) (err error) {
	defer func() { err = e.WrapIfNil("can't remove game", err) }()

	user, err := p.storage.GetUserByName(ctx, userName)
	if err != nil && !errors.Is(err, storage.ErrNoUser) {
		return err
	}
	if errors.Is(err, storage.ErrNoUser) {
		return p.tg.SendMessage(ctx, chatId, msgNoWishlist)
	}

	wishlist, err := p.storage.GetAll(ctx, user)
	if err != nil && !errors.Is(err, storage.ErrNoWishlist) {
		return err
	}
	if errors.Is(err, storage.ErrNoWishlist) {
		return p.tg.SendMessage(ctx, chatId, msgNoWishlist)
	}

	var buttons [][]telegram.InlineKeyboardButton

	for _, w := range wishlist {
		button := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("💀%s", w.Game.Name),
			CallbackData: fmt.Sprintf("remove:%d", w.Id),
		}

		buttons = append(buttons, []telegram.InlineKeyboardButton{button})
	}

	return p.tg.SendMessageWithKeyboard(ctx, chatId, msgRemoveGameChoice, &telegram.InlineKeyboardMarkup{InlineKeyboard: buttons})
}

func (p *Processor) sendGameList(ctx context.Context, chatId int, userName string) (err error) {
	defer func() { err = e.WrapIfNil("can't send game list", err) }()

	user, err := p.storage.GetUserByName(ctx, userName)
	if err != nil && !errors.Is(err, storage.ErrNoUser) {
		return err
	}
	if errors.Is(err, storage.ErrNoUser) {
		return p.tg.SendMessage(ctx, chatId, msgNoWishlist)
	}

	wishlist, err := p.storage.GetAll(ctx, user)
	if err != nil && !errors.Is(err, storage.ErrNoWishlist) {
		return err
	}
	if errors.Is(err, storage.ErrNoWishlist) {
		return p.tg.SendMessage(ctx, chatId, msgNoWishlist)
	}

	var builder strings.Builder
	builder.WriteString(msgGameList)

	for _, w := range wishlist {
		builder.WriteString(fmt.Sprintf("\n\n🎯 %s\n", w.Game.Name))
		if !w.ExpectedReleaseDate.IsZero() {
			builder.WriteString(fmt.Sprintf("🔜 Дата релиза: %s\n", w.ExpectedReleaseDate.Format("02.01.2006")))
		}
		builder.WriteString(fmt.Sprintf("🔗 %s", w.Game.ExternalURL))
	}

	return p.tg.SendMessage(ctx, chatId, builder.String())
}

func (p *Processor) searchGameList(ctx context.Context, text string, chatID int) (err error) {
	defer func() { err = e.WrapIfNil("can't search game", err) }()

	var res []api.SearchResult
	res, err = p.finder.Find(ctx, text)
	if err != nil && !errors.Is(err, api.ErrNoSearchResults) {
		return err
	}
	if errors.Is(err, api.ErrNoSearchResults) {
		return p.tg.SendMessage(ctx, chatID, msgNoSearchResults)
	}

	var buttons [][]telegram.InlineKeyboardButton

	for _, game := range res {
		buttonText := fmt.Sprintf("🎮 %s", game.Name)
		if !game.FirstReleaseDate.IsZero() {
			buttonText += " (" + strconv.Itoa(game.FirstReleaseDate.Year()) + ")"
		}
		button := telegram.InlineKeyboardButton{
			Text:         buttonText,
			CallbackData: fmt.Sprintf("select:%d", game.Id),
		}
		buttons = append(buttons, []telegram.InlineKeyboardButton{button})
	}

	return p.tg.SendMessageWithKeyboard(ctx, chatID, msgGameListChoice, &telegram.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	})
}

func (p *Processor) sendHelp(ctx context.Context, chatId int) error {
	return p.tg.SendMessage(ctx, chatId, msgHelp)
}

func (p *Processor) sendHello(ctx context.Context, chatId int) error {
	return p.tg.SendMessage(ctx, chatId, msgHello)
}
