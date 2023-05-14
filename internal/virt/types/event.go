package types

import "time"

type Event struct {
	ID     string
	Type   EventType
	Action EventAction
	Time   time.Time
}

var EventTypeGuest EventType = "guest"

type EventType string

var (
	ActionCreate   EventAction = "create"
	ActionBoot     EventAction = "boot"
	ActionShutdown EventAction = "shutdown"
	ActionDelete   EventAction = "delete"
)

type EventAction string
