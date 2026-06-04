package selector

import "time"

type Event struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt *time.Time
}
