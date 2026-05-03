package audit

import "context"

type Auditor struct {
	observers []Observer
}

func NewAuditor(observers ...Observer) *Auditor {
	return &Auditor{
		observers: observers,
	}
}

func (a *Auditor) Enabled() bool {
	return len(a.observers) > 0
}

func (a *Auditor) Notify(ctx context.Context, event Event) {
	for _, observer := range a.observers {
		_ = observer.Notify(ctx, event)
	}
}
