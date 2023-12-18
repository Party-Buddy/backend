package converters

import (
	"party-buddy/internal/configuration"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/utils"
)

func ToMessageTaskStart(m session.MsgTaskStart) ws.MessageTaskStart {
	msg := ws.MessageTaskStart{
		BaseMessage: utils.GenBaseMessage(&ws.MsgKindTaskStart),
		TaskIdx:     uint8(m.TaskIdx),
		Deadline:    ws.Time(m.Deadline),
	}
	if m.Options != nil {
		msg.Options = m.Options
		return msg
	}
	if m.ImgID != nil {
		uri := configuration.GenImgURI(m.ImgID.UUID)
		msg.ImgURI = &uri
	}
	return msg
}
