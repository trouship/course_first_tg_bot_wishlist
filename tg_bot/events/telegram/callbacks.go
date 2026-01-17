package telegram

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"tg_game_wishlist/api"
	"tg_game_wishlist/clients/telegram"
	"tg_game_wishlist/lib/e"
	"tg_game_wishlist/storage"
	"time"
)

const (
	SelectCallback = "select"
	AddCallback    = "add"
	RemoveCallback = "remove"
	AddWithoutDate = "add_without_date"
)

func (p *Processor) doCallback(ctx context.Context, callbackId string, text string, chatID int, userName string) (err error) {
	defer func() { err = e.WrapIfNil("can't process callback", err) }()

	parts := strings.Split(text, ":")

	switch parts[0] {
	case SelectCallback:
		return p.selectGameCallback(ctx, callbackId, text, chatID, userName)
	case AddCallback:
		return p.addGameCallback(ctx, callbackId, text, chatID, userName)
	case RemoveCallback:
		return p.removeWishlistCallback(ctx, callbackId, text, chatID)
	case AddWithoutDate:
		return p.addWithoutDateCallback(ctx, callbackId, chatID, userName)
	}

	return nil
}

func (p *Processor) addWithoutDateCallback(ctx context.Context, callbackId string, chatId int, userName string) (err error) {
	defer func() {
		err = e.WrapIfNil("can't add game without date callback", err)
		p.tg.AnswerCallBack(ctx, callbackId, "", false)
	}()

	state, ok := p.states[userName]
	if !ok {
		return err
	}

	return p.addManualGame(ctx, chatId, userName, state.GameName, time.Time{})
}

func (p *Processor) removeWishlistCallback(ctx context.Context, callbackId string, text string, chatID int) (err error) {
	defer func() {
		err = e.WrapIfNil("can't process remove wishlist callback", err)
		p.tg.AnswerCallBack(ctx, callbackId, "", false)
	}()
	parts := strings.Split(text, ":")

	wishlistId, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	if err = p.storage.Remove(ctx, wishlistId); err != nil {
		return err
	}

	return p.tg.SendMessage(ctx, chatID, msgRemoved)
}

func (p *Processor) addGameCallback(ctx context.Context, callbackId string, text string, chatID int, userName string) (err error) {
	defer func() {
		err = e.WrapIfNil("can't process add game callback", err)
		p.tg.AnswerCallBack(ctx, callbackId, "", false)
	}()
	parts := strings.Split(text, ":")

	gameId, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	searchGame, err := p.gameById(ctx, gameId)
	if err != nil {
		return err
	}

	//Если не указана платформа
	if len(parts) < 3 {
		return p.addApiGame(ctx, searchGame, nil, chatID, userName)
	}

	platformIds := strings.Split(parts[2], ",")
	for _, rd := range searchGame.ReleaseDates {
		if slices.Contains(platformIds, strconv.Itoa(rd.Platform.Id)) {
			return p.addApiGame(ctx, searchGame, &rd, chatID, userName)
		}
	}

	return e.Wrap("platform not found", err)
}

func (p *Processor) selectGameCallback(ctx context.Context, callbackId string, text string, chatID int, userName string) (err error) {
	defer func() {
		err = e.WrapIfNil("can't process select game callback", err)
		p.tg.AnswerCallBack(ctx, callbackId, "", false)
	}()

	parts := strings.Split(text, ":")

	gameId, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	searchGame, err := p.gameById(ctx, gameId)
	if err != nil {
		return err
	}

	now := time.Now()

	//Случаи когда действия от пользователя не требуются
	if p.isPastDates(searchGame.ReleaseDates) {
		//Случай с всеми прошедшими датами (просто добавление без даты)
		return p.addApiGame(ctx, searchGame, nil, chatID, userName)
	} else if p.isSameDatePlatform(searchGame.ReleaseDates) {
		//Случай с одинаковыми датами у всех платформ
		if len(searchGame.ReleaseDates) > 0 && searchGame.ReleaseDates[0].Date.After(now) {
			//Если дата в будущем, то добавляем с датой
			return p.addApiGame(ctx, searchGame, &searchGame.ReleaseDates[0], chatID, userName)
		} else {
			//Если даты нет или она в прошлом, то добавление без даты
			return p.addApiGame(ctx, searchGame, nil, chatID, userName)
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

		var ids []string
		var names []string

		for _, platform := range platforms {
			ids = append(ids, strconv.Itoa(platform.Id))
			names = append(names, platform.Name)
		}
		buttonText := strings.Join(names, " | ")
		buttonCallbackIds := strings.Join(ids, ",")
		button := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s 📅 %s", buttonText, date.Format("02.01.2006")),
			CallbackData: fmt.Sprintf("add:%d:%s", searchGame.Id, buttonCallbackIds),
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

	if err = p.tg.SendMessageWithKeyboard(ctx, chatID, msgPlatformDateChoice, &telegram.InlineKeyboardMarkup{InlineKeyboard: buttons}); err != nil {
		return err
	}

	return nil
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

func (p *Processor) addApiGame(ctx context.Context, searchGame *api.Game, platformDate *api.PlatformDate, chatID int, userName string) (err error) {
	defer func() {
		err = e.WrapIfNil("can't add api game to storage", err)
	}()

	user := &storage.User{
		Name: userName,
	}

	game := &storage.Game{
		Name:        searchGame.Name,
		Source:      searchGame.Source,
		ExternalURL: searchGame.URL,
	}

	wishlist := &storage.Wishlist{
		User: user,
		Game: game,
	}
	if platformDate != nil {
		wishlist.ExpectedReleaseDate = platformDate.Date
	}

	isExists, err := p.storage.IsExists(ctx, wishlist)
	if err != nil {
		return err
	}
	if isExists {
		return p.tg.SendMessage(ctx, chatID, msgAlreadyExists)
	}

	if err := p.storage.Add(ctx, wishlist); err != nil {
		return err
	}

	return p.tg.SendMessage(ctx, chatID, msgSaved)
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
