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
	"time"
)

const (
	HelpCmd   = "/help"
	StartCmd  = "/start"
	ListCmd   = "/list"
	RemoveCmd = "/remove"
)

func (p *Processor) doCmd(ctx context.Context, text string, chatID int, userName string) error {
	//text = strings.TrimSpace(text)

	log.Printf("got new command '%s' from '%s'", text, userName)

	state, ok := p.states[userName]

	if ok && state.Step == AwaitingDateStep {
		//Если не дата, то очищаем состояние
		date, err := p.parseDateFromString(text)
		if err != nil {
			p.clearState(userName)
		} else {
			//Иначе проверяем дату

			//Если в прошлом, то не добавляем
			if date.Before(time.Now()) {
				return p.tg.SendMessage(ctx, chatID, msgPreviousDate)
			}

			//Иначе добавляем с датой
			return p.addManualGameWithDate(ctx, chatID, userName, state.GameName, date)
		}
	}

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
		return p.searchGameList(ctx, text, chatID, userName)
	}
}

func (p *Processor) parseDateFromString(strDate string) (time.Time, error) {
	return time.Parse("02.01.2006", strDate)
}

func (p *Processor) addManualGameWithDate(ctx context.Context, chatId int, userName string, gameName string, date time.Time) (err error) {
	defer func() { err = e.WrapIfNil("can't add manual", err) }()

	return p.addManualGame(ctx, chatId, userName, gameName, date)
}

func (p *Processor) clearState(userName string) {
	delete(p.states, userName)
}

func (p *Processor) addManualGame(ctx context.Context, chatId int, userName string, gameName string, date time.Time) (err error) {
	defer func() { err = e.WrapIfNil("can't add manual game to storage", err) }()

	user := &storage.User{
		Name:   userName,
		ChatId: chatId,
	}

	game := &storage.Game{
		Name:   gameName,
		Source: storage.Manual,
	}

	wishlist := &storage.Wishlist{
		User: user,
		Game: game,
	}
	if !date.IsZero() {
		wishlist.NotificationDate = date
	}

	isExists, err := p.storage.IsExists(ctx, wishlist)
	if err != nil {
		return err
	}
	if isExists {
		return p.tg.SendMessage(ctx, chatId, msgAlreadyExists)
	}

	if err := p.storage.Add(ctx, wishlist); err != nil {
		return err
	}

	p.clearState(userName)

	return p.tg.SendMessage(ctx, chatId, msgSaved)
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
		builder.WriteString(fmt.Sprintf("\n\n🎯 %s", w.Game.Name))
		if !w.NotificationDate.IsZero() {
			builder.WriteString(fmt.Sprintf("\n🔔 Дата уведомления: %s", w.NotificationDate.Format("02.01.2006")))
		}
		if w.Game.ExternalURL != "" {
			builder.WriteString(fmt.Sprintf("\n🔗 %s", w.Game.ExternalURL))
		}
	}

	return p.tg.SendMessage(ctx, chatId, builder.String())
}

func (p *Processor) searchGameList(ctx context.Context, text string, chatID int, userName string) (err error) {
	defer func() { err = e.WrapIfNil("can't search game", err) }()

	var res []api.SearchResult
	res, err = p.finder.Find(ctx, text)
	if err != nil && !errors.Is(err, api.ErrNoSearchResults) {
		return err
	}
	if errors.Is(err, api.ErrNoSearchResults) {
		return p.sendNoSearchResults(ctx, text, chatID, userName)
	}

	var buttons [][]telegram.InlineKeyboardButton

	for _, game := range res {
		buttonText := fmt.Sprintf("🎮 %s", game.Name)
		if !game.FirstReleaseDate.IsZero() {
			buttonText += fmt.Sprintf(" (%s)", strconv.Itoa(game.FirstReleaseDate.Year()))
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

func (p *Processor) sendNoSearchResults(ctx context.Context, text string, chatId int, userName string) (err error) {
	defer func() { err = e.WrapIfNil("can't send no search results", err) }()

	p.states[userName] = &UserState{
		GameName: text,
		Step:     AwaitingDateStep,
	}

	var buttons [][]telegram.InlineKeyboardButton

	button := telegram.InlineKeyboardButton{
		Text:         btnAddGameWithoutDate,
		CallbackData: "add_without_date",
	}
	buttons = append(buttons, []telegram.InlineKeyboardButton{button})

	return p.tg.SendMessageWithKeyboard(ctx, chatId, msgNoSearchResults, &telegram.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	})
}

func (p *Processor) sendHelp(ctx context.Context, chatId int) error {
	return p.tg.SendMessage(ctx, chatId, msgHelp)
}

func (p *Processor) sendHello(ctx context.Context, chatId int) error {
	return p.tg.SendMessage(ctx, chatId, msgHello)
}
