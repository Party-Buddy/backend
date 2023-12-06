package converters

import (
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/utils"
)

func ToMessageGameStatus(m session.MsgGameStatus) ws.MessageGameStatus {
	var players []ws.Player

	for _, player := range m.Players {
		players = append(players, ws.Player{
			PlayerID: player.ID.UUID(),
			Nickname: player.Nickname,
		})
	}

	return ws.MessageGameStatus{
		BaseMessage: utils.GenBaseMessage(&ws.MsgKindGameStatus),
		Players:     players,
	}
}
