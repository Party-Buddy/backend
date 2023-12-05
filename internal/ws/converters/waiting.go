package converters

import (
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/utils"

	"github.com/google/uuid"
)

func ToMessageWaiting(m session.MsgWaiting) ws.MessageWaiting {
	var ready []uuid.UUID

	for playerID := range m.PlayersReady {
		ready = append(ready, playerID.UUID())
	}

	return ws.MessageWaiting{
		BaseMessage: utils.GenBaseMessage(&ws.MsgKindWaiting),
		Ready:       ready,
	}
}
