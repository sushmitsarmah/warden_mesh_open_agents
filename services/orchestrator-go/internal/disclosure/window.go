package disclosure

import (
	"time"
)

type Window struct {
	publishedAt time.Time
	revealAt    time.Time
}

func NewWindow(publishedAt time.Time, days int) *Window {
	return &Window{
		publishedAt: publishedAt,
		revealAt:    publishedAt.Add(time.Duration(days) * 24 * time.Hour),
	}
}

func (w *Window) CanRevealPublicly() bool {
	return time.Now().After(w.revealAt)
}

func (w *Window) IsPaidAccess() bool {
	return true // protocol pays to unlock before window
}
