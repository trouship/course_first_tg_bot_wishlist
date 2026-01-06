package event_consumer

import (
	"context"
	"log"
	"tg_game_wishlist/events"
	"time"
)

type Consumer struct {
	fetcher   events.Fetcher
	processor events.Processor
	batchSize int
	timeout   int
}

func New(fetcher events.Fetcher, processor events.Processor, batchSize int, timeout int) Consumer {
	return Consumer{
		fetcher:   fetcher,
		processor: processor,
		batchSize: batchSize,
		timeout:   timeout,
	}
}

func (c Consumer) Start() error {
	for {
		gotEvents, err := c.fetcher.Fetch(context.Background(), c.batchSize, c.timeout)
		if err != nil {
			log.Printf("[ERR] consumer: %s", err.Error())
			time.Sleep(3 * time.Second)
			continue
		}

		if len(gotEvents) == 0 {
			continue
		}

		if err := c.handleEvents(context.Background(), gotEvents); err != nil {
			log.Print(err)
			continue
		}
	}
}

func (c Consumer) handleEvents(ctx context.Context, events []events.Event) error {
	for _, event := range events {
		log.Printf("got new event: %s", event.Text)

		if err := c.processor.Process(ctx, event); err != nil {
			log.Printf("can't handle event: %s", err.Error())
			continue
		}
	}

	return nil
}
