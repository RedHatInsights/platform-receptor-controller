package controller

import (
	"github.com/google/uuid"
)

type Message struct {
	MessageID uuid.UUID
	Recipient string
	RouteList []string
	Payload   interface{}
	Directive string
}
