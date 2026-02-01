package telegram

import (
	"context"
	"log"
	"strings"
	"tg_game_wishlist/clients/telegram"
	"tg_game_wishlist/notifier"
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
					continue
				}

				if len(wishlist) == 0 {
					continue
				}

				//Группировка списка желаемого по пользователям
				userWishlist := make(map[int][]storage.Wishlist)
				for _, w := range wishlist {
					userWishlist[w.User.ChatId] = append(userWishlist[w.User.ChatId], w)
				}

				for chatId, uw := range userWishlist {
					var builder strings.Builder
					builder.WriteString(notifier.MsgTodayGameReleases)

					for _, w := range uw {
						builder.WriteString("\n\n")
						builder.WriteString("🔥 ")
						builder.WriteString(w.Game.Name)

						if w.Game.ExternalURL != "" {
							builder.WriteString("\n🌐 ")
							builder.WriteString(w.Game.ExternalURL)
						}
					}

					err = n.tg.SendMessage(ctx, chatId, builder.String())
					if err != nil {
						log.Printf("[ERR] can't send notification: %w", err)
						continue
					}

					for _, w := range uw {
						err = n.storage.Notify(ctx, &w)
						if err != nil {
							log.Printf("[ERR] can't storage notify: %w", err)
						}
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
