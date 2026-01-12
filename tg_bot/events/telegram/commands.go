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
	"tg_game_wishlist/storage"
	"time"
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

	gameId, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	game, err := p.gameById(ctx, gameId)
	if err != nil {
		return err
	}

	switch parts[0] {
	case "select":
		return p.selectGameCallback(ctx, callbackId, game, chatID, userName)
	case "add":

	}

	return nil
}

func (p *Processor) selectGameCallback(ctx context.Context, callbackId string, searchGame *api.Game, chatID int, userName string) (err error) {
	defer func() { err = e.WrapIfNil("can't process select game callback", err) }()

	now := time.Now()
	p.tg.AnswerCallBack(ctx, callbackId, "test", false)

	//Случаи когда действия от пользователя не требуются
	if p.isPastDates(searchGame.ReleaseDates) {
		//Случай с всеми прошедшими датами (просто добавление без даты)
		return p.addGame(ctx, callbackId, searchGame, nil, chatID, userName)
	} else if p.isSameDatePlatform(searchGame.ReleaseDates) {
		//Случай с одинаковыми датами у всех платформ
		if len(searchGame.ReleaseDates) > 0 && searchGame.ReleaseDates[0].Date.After(now) {
			//Если дата в будущем, то добавляем с датой
			return p.addGame(ctx, callbackId, searchGame, &searchGame.ReleaseDates[0], chatID, userName)
		} else {
			//Если даты нет или она в прошлом, то добавление без даты
			return p.addGame(ctx, callbackId, searchGame, nil, chatID, userName)
		}
	}

	//Случай, когда даты разные, и как минимум одна из них в будущем
	var buttons [][]telegram.InlineKeyboardButton

	grouped := p.groupGamePlatformsByDate(searchGame.ReleaseDates)
	var oldDatePlatforms []string
	for date, platforms := range grouped {
		if now.After(date) {
			for _, platform := range platforms {
				oldDatePlatforms = append(oldDatePlatforms, platform.Name)
			}
			continue
		}

		var names []string

		for _, platform := range platforms {
			names = append(names, platform.Name)
		}
		buttonText := strings.Join(names, " | ")
		//TODO добавить перечисление нескольких платформ через запятую
		button := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s\n 📅 %s", buttonText, date.Format("02.01.2006")),
			CallbackData: fmt.Sprintf("add:%d:%s", searchGame.Id, names),
		}
		buttons = append(buttons, []telegram.InlineKeyboardButton{button})
	}

	if len(oldDatePlatforms) > 0 {
		button := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s", strings.Join(oldDatePlatforms, " | ")+" (Ранее)"),
			CallbackData: fmt.Sprintf("add:%d", searchGame.Id),
		}
		buttons = append(buttons, []telegram.InlineKeyboardButton{button})
	}

	if err = p.tg.SendMessageWithKeyboard(ctx, chatID, "Платформы:", &telegram.InlineKeyboardMarkup{InlineKeyboard: buttons}); err != nil {
		return err
	}

	return p.tg.AnswerCallBack(ctx, callbackId, "", false)
}

func (p *Processor) groupGamePlatformsByDate(releaseDates []api.PlatformDate) map[time.Time][]api.Platform {
	res := make(map[time.Time][]api.Platform)

	for _, platformDate := range releaseDates {
		day := platformDate.Date.Truncate(24 * time.Hour)
		platform := api.Platform{
			Id:   platformDate.Platform.Id,
			Name: platformDate.Platform.Name,
		}
		res[day] = append(res[day], platform)
	}

	return res
}

func (p *Processor) addGame(ctx context.Context, callbackId string, searchGame *api.Game, platformDate *api.PlatformDate, chatID int, userName string) (err error) {
	defer func() { err = e.WrapIfNil("can't add game to storage", err) }()
	return nil

	user := &storage.User{
		Name: userName,
	}

	game := &storage.Game{
		Name:        searchGame.Name,
		Source:      searchGame.Source,
		ExternalURL: searchGame.URL,
	}
	if platformDate != nil {
		game.ReleaseDate = platformDate.Date
	}

	wishlist := &storage.Wishlist{
		User: user,
		Game: game,
	}

	isExists, err := p.storage.IsExists(ctx, wishlist)
	if err != nil {
		return err
	}
	if isExists {
		//TODO change message
		return p.tg.SendMessage(ctx, chatID, "exists")
	}

	if err := p.storage.Add(ctx, wishlist); err != nil {
		return err
	}

	//TODO change message
	if err := p.tg.SendMessage(ctx, chatID, "success"); err != nil {
		return err
	}

	return p.tg.AnswerCallBack(ctx, callbackId, "", false)
}

func (p *Processor) isSameDatePlatform(platformDates []api.PlatformDate) bool {
	if len(platformDates) <= 1 {
		return true
	}

	firstDate := platformDates[0].Date
	for _, date := range platformDates[1:] {
		if !date.Date.Equal(firstDate) {
			return false
		}
	}

	return true
}

func (p *Processor) isPastDates(platformDates []api.PlatformDate) bool {
	now := time.Now()

	for _, date := range platformDates {
		if date.Date.After(now) {
			return false
		}
	}

	return true
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

	return p.tg.SendMessageWithKeyboard(ctx, chatID, "Выберите игру: ", &telegram.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	})
}
