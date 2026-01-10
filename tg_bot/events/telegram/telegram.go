package telegram

import (
	"context"
	"errors"
	"tg_game_wishlist/api"
	"tg_game_wishlist/clients/telegram"
	"tg_game_wishlist/events"
	"tg_game_wishlist/lib/e"
	"tg_game_wishlist/storage"
)

type Processor struct {
	tg      *telegram.Client
	finder  api.Finder
	storage storage.Storage
}

type Fetcher struct {
	tg     *telegram.Client
	offset int
}

type Meta struct {
	ChatId   int
	UserName string
}

var (
	ErrUnknownEventType = errors.New("unknown event type")
	ErrUnknownMetaType  = errors.New("unknown meta type")
)

func NewProcessor(client *telegram.Client, finder api.Finder, storage storage.Storage) *Processor {
	return &Processor{
		tg:      client,
		finder:  finder,
		storage: storage,
	}
}

func NewFetcher(client *telegram.Client) *Fetcher {
	return &Fetcher{
		tg: client,
	}
}

func (p *Processor) Process(ctx context.Context, event events.Event) error {
	switch event.Type {
	case events.Message:
		return p.processMessage(ctx, event)
	case events.CallbackQuery:
		return p.processCallbackQuery(ctx, event)
	default:
		return e.Wrap("can't process event", ErrUnknownEventType)
	}
}

func (p *Processor) processMessage(ctx context.Context, event events.Event) error {
	meta, err := meta(event)
	if err != nil {
		return e.Wrap("can't process message", err)
	}

	if err := p.doCmd(ctx, event.Text, meta.ChatId, meta.UserName); err != nil {
		return e.Wrap("can't process message", err)
	}

	return nil
}

func (p *Processor) processCallbackQuery(ctx context.Context, event events.Event) error {
	meta, err := meta(event)
	if err != nil {
		return e.Wrap("can't process callback query", err)
	}

	if err := p.doCallback(ctx, event.Id, event.Text, meta.ChatId, meta.UserName); err != nil {
		return e.Wrap("can't process callback query", err)
	}

	return nil
}

func meta(event events.Event) (Meta, error) {
	res, ok := event.Meta.(Meta)
	if !ok {
		return Meta{}, e.Wrap("can't get meta", ErrUnknownMetaType)
	}

	return res, nil
}

func (f *Fetcher) Fetch(ctx context.Context, limit int, timeout int) ([]events.Event, error) {
	updates, err := f.tg.Updates(ctx, f.offset, limit, timeout)
	if err != nil {
		return nil, e.Wrap("can't get events", err)
	}

	if len(updates) == 0 {
		return nil, nil
	}

	res := make([]events.Event, 0, len(updates))

	for _, u := range updates {
		res = append(res, event(u))
	}

	f.offset = updates[len(updates)-1].Id + 1

	return res, nil
}

func event(upd telegram.Update) events.Event {
	updType := fetchType(upd)

	res := events.Event{
		Id:   fetchId(upd),
		Type: updType,
		Text: fetchText(upd),
	}

	switch updType {
	case events.Message:
		res.Meta = Meta{
			ChatId:   upd.Message.Chat.Id,
			UserName: upd.Message.From.Username,
		}
	case events.CallbackQuery:
		res.Meta = Meta{
			ChatId:   upd.CallbackQuery.Message.Chat.Id,
			UserName: upd.CallbackQuery.From.Username,
		}
	case events.Unknown:
		res.Meta = nil
	}

	return res
}

func fetchId(upd telegram.Update) string {
	if upd.CallbackQuery == nil {
		return ""
	}

	return upd.CallbackQuery.Id
}

func fetchText(upd telegram.Update) string {
	if upd.CallbackQuery != nil {
		return upd.CallbackQuery.Data
	} else if upd.Message == nil {
		return ""
	}

	return upd.Message.Text
}

func fetchType(upd telegram.Update) events.Type {
	if upd.CallbackQuery != nil {
		return events.CallbackQuery
	} else if upd.Message == nil {
		return events.Unknown
	}

	return events.Message
}
