package audit

import "context"

// Auditor dispatches metric audit events to configured observers.
type Auditor struct {
	observers []Observer
}

// NewAuditor creates an Auditor with the provided observers.
func NewAuditor(observers ...Observer) *Auditor {
	return &Auditor{
		observers: observers,
	}
}

// Enabled reports whether the auditor has observers.
func (a *Auditor) Enabled() bool {
	return len(a.observers) > 0
}

// Notify sends an audit event to all configured observers.
func (a *Auditor) Notify(ctx context.Context, event Event) {
	for _, observer := range a.observers {
		_ = observer.Notify(ctx, event)
	}
}
