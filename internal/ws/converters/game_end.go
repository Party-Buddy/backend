package converters

import (
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/ws/utils"
)

func ToMessageGameEnd(m session.MsgGameEnd) ws.MessageGameEnd {
	var scores []ws.GamePlayerScore

	for _, score := range m.Scoreboard.Scores() {
		scores = append(scores, ws.GamePlayerScore{
			PlayerID:    uint32(score.PlayerID),
			TotalPoints: uint32(score.Score),
		})
	}

	return ws.MessageGameEnd{
		BaseMessage: utils.GenBaseMessage(&ws.MsgKindGameEnd),
		Scoreboard:  scores,
	}
}
