package audit

import "context"

type Observer interface {
	Notify(ctx context.Context, event Event) error
}
