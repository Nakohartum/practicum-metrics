package audit

import "context"

// Observer receives audit events.
type Observer interface {
	Notify(ctx context.Context, event Event) error
}
