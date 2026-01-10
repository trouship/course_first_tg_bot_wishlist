package events

import "context"

type Fetcher interface {
	Fetch(ctx context.Context, limit int, timeout int) ([]Event, error)
}

type Processor interface {
	Process(ctx context.Context, e Event) error
}

type Type int

const (
	Unknown = iota
	Message
	CallbackQuery
)

type Event struct {
	Id   string
	Type Type
	Text string
	Meta interface{}
}
