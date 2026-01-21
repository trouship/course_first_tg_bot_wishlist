package notifier

import "context"

type Notifier interface {
	Start(ctx context.Context) error
}
