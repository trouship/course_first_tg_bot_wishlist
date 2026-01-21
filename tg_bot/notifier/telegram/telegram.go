package telegram

import (
	"context"
	"log"
	"tg_game_wishlist/clients/telegram"
	"tg_game_wishlist/storage"
	"time"
)

type Notifier struct {
	storage  storage.Storage
	tg       *telegram.Client
	interval time.Duration
}

func New(storage storage.Storage, tg *telegram.Client, duration time.Duration) *Notifier {
	return &Notifier{
		storage:  storage,
		tg:       tg,
		interval: duration,
	}
}

func (n *Notifier) Start(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(n.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				wishlist, err := n.storage.GetToNotify(ctx)
				if err != nil {
					log.Printf("[ERR] can't get wishlists to notify: %w", err)
				}

				for _, w := range wishlist {
					err = n.tg.SendMessage(ctx, w.User.ChatId, w.Game.Name)
					if err != nil {
						log.Printf("[ERR] can't send notification: %w", err)
					}

					err = n.storage.Notify(ctx, &w)
					if err != nil {
						log.Printf("[ERR] can't storage notify: %w", err)
					}
				}

			case <-ctx.Done():
				log.Println("Notifier stopped")
				return
			}

		}
	}()

	return nil
}
