package session

import (
	"context"
	"time"

	"golang.org/x/exp/maps"
)

func (m *Manager) makeMsgError(ctx context.Context, err error) ServerTx {
	return &MsgError{
		baseTx: baseTx{Ctx: ctx},
		Inner:  err,
	}
}

func (m *Manager) makeMsgJoined(
	ctx context.Context,
	playerID PlayerID,
	sid SessionID,
	inviteCode *InviteCode,
	game *Game,
	maxPlayers int,
) ServerTx {
	return &MsgJoined{
		baseTx:     baseTx{Ctx: ctx},
		PlayerID:   playerID,
		SessionID:  sid,
		InviteCode: inviteCode,
		Game:       game,
		MaxPlayers: maxPlayers,
	}
}

func (m *Manager) makeMsgGameStatus(ctx context.Context, players []Player) ServerTx {
	return &MsgGameStatus{
		baseTx:  baseTx{Ctx: ctx},
		Players: players,
	}
}

func (m *Manager) makeMsgTaskStart(
	ctx context.Context,
	taskIdx int,
	deadline time.Time,
	task Task,
	answer TaskAnswer,
) ServerTx {
	msg := &MsgTaskStart{
		baseTx:   baseTx{Ctx: ctx},
		TaskIdx:  taskIdx,
		Deadline: deadline,
	}
	switch t := task.(type) {
	case ChoiceTask:
		msg.Options = &t.Options
		return msg
	case PhotoTask:
		i := ImageID(answer.(PhotoTaskAnswer))
		msg.ImgID = &i
		return msg
	default:
		return msg
	}

}

func (m *Manager) makeMsgPollStart(
	ctx context.Context,
	taskIdx int,
	deadline time.Time,
	options []PollOption,
) ServerTx {
	// TODO
	return nil
}

func (m *Manager) makeMsgTaskEnd(
	ctx context.Context,
	taskIdx int,
	deadline time.Time,
	task Task,
	scoreboard Scoreboard,
	winners map[PlayerID]Score,
	results []AnswerResult,
) ServerTx {
	return &MsgTaskEnd{
		baseTx:     baseTx{Ctx: ctx},
		TaskIdx:    taskIdx,
		Deadline:   deadline,
		Task:       task,
		Results:    results,
		Scoreboard: scoreboard,
		Winners:    winners,
	}
}

func (m *Manager) makeMsgGameEnd(ctx context.Context, scoreboard Scoreboard) ServerTx {
	return &MsgGameEnd{
		baseTx:     baseTx{Ctx: ctx},
		Scoreboard: scoreboard,
	}
}

func (m *Manager) makeMsgGameStart(ctx context.Context, deadline time.Time) ServerTx {
	return &MsgGameStart{
		baseTx:   baseTx{Ctx: ctx},
		Deadline: deadline,
	}
}

func (m *Manager) makeMsgWaiting(ctx context.Context, playersReady map[PlayerID]struct{}) ServerTx {
	return &MsgWaiting{
		baseTx:       baseTx{Ctx: ctx},
		PlayersReady: maps.Clone(playersReady),
	}
}
