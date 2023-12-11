package converters

import (
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/utils"
)

func ToMessageGameStart(m session.MsgGameStart) ws.MessageGameStart {
	return ws.MessageGameStart{
		BaseMessage: utils.GenBaseMessage(&ws.MsgKindGameStart),
		// TODO: time delay compensation
		Deadline: ws.Time(m.Deadline),
	}
}
