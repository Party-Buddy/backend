package converters

import (
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/utils"
)

func ToMessageWaiting(m session.MsgWaiting) ws.MessageWaiting {
	var ready []uint32

	for playerID := range m.PlayersReady {
		ready = append(ready, uint32(playerID))
	}

	return ws.MessageWaiting{
		BaseMessage: utils.GenBaseMessage(&ws.MsgKindWaiting),
		Ready:       ready,
	}
}
