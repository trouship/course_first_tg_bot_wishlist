package telegram

import (
	"context"
	"tg_game_wishlist/clients/telegram"
	"tg_game_wishlist/events"
	"tg_game_wishlist/lib/e"
)

type Processor struct {
	tg     *telegram.Client
	offset int
	//TODO storage storage.Storage
}

type Meta struct {
	ChatId   int
	UserName string
}

func (p *Processor) Fetch(ctx context.Context, limit int, timeout int) ([]events.Event, error) {
	updates, err := p.tg.Updates(ctx, p.offset, limit, timeout)
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

	p.offset = updates[len(updates)-1].Id + 1

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
