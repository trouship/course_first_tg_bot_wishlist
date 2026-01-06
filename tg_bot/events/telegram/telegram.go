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
	finder  *api.Finder
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

func New(client *telegram.Client, finder *api.Finder, storage storage.Storage) *Processor {
	return &Processor{
		tg:      client,
		finder:  finder,
		storage: storage,
	}
}

func (p *Processor) Process(ctx context.Context, event events.Event) error {
	switch event.Type {
	case events.Message:
		return p.processMessage(ctx, event)
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
		Type: updType,
		Text: fetchText(upd),
	}

	if updType == events.Message {
		res.Meta = Meta{
			ChatId:   upd.Message.Chat.Id,
			UserName: upd.Message.From.Username,
		}
	}

	return res
}

func fetchText(upd telegram.Update) string {
	if upd.Message == nil {
		return ""
	}

	return upd.Message.Text
}

func fetchType(upd telegram.Update) events.Type {
	if upd.Message == nil {
		return events.Unknown
	}

	return events.Message
}
