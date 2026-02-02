package notifier

import "context"

const (
	MsgTodayGameReleases = "📢 Сегодня выходят:"
)

type Notifier interface {
	Start(ctx context.Context) error
}
