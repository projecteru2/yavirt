package types

import "time"

type Event struct {
	ID     string
	Type   string
	Action string
	Time   time.Time
}
