package ws

import "party-buddy/internal/schemas/ws"

type sessionState interface {
	isSessionState()
	isAllowedMsg(m ws.RecvMessage) bool
	name() string
}

type initialState struct{}

func (initialState) isSessionState() {}

func (initialState) isAllowedMsg(m ws.RecvMessage) bool {
	switch m.(type) {
	case *ws.MessageJoin:
		return true
	default:
		return false
	}
}

func (initialState) name() string {
	return "initial"
}

type awaitingPlayersState struct{}

func (awaitingPlayersState) isSessionState() {}

func (awaitingPlayersState) isAllowedMsg(m ws.RecvMessage) bool {
	switch m.(type) {
	case *ws.MessageReady:
		return true
	case *ws.MessageLeave:
		return true
	case *ws.MessageKick:
		return true
	default:
		return false
	}
}

func (awaitingPlayersState) name() string {
	return "awaiting-players"
}

type gameStartedState struct{}

func (gameStartedState) isSessionState() {}

func (gameStartedState) isAllowedMsg(m ws.RecvMessage) bool {
	switch m.(type) {
	case *ws.MessageReady:
		return true
	case *ws.MessageLeave:
		return true
	case *ws.MessageKick:
		return true
	default:
		return false
	}
}

func (gameStartedState) name() string {
	return "game-started"
}

type taskStartedState struct{}

func (taskStartedState) isSessionState() {}

func (taskStartedState) isAllowedMsg(m ws.RecvMessage) bool {
	switch m.(type) {
	case *ws.MessageReady:
		return true
	case *ws.MessageLeave:
		return true
	case *ws.MessageKick:
		return true
	case *ws.MessageTaskAnswer:
		return true
	case *ws.MessagePollChoose:
		return true
	default:
		return false
	}
}

func (taskStartedState) name() string {
	return "task-started"
}

type pollStartedState struct{}

func (pollStartedState) isSessionState() {}

func (pollStartedState) isAllowedMsg(m ws.RecvMessage) bool {
	switch m.(type) {
	case *ws.MessageReady:
		return true
	case *ws.MessageLeave:
		return true
	case *ws.MessageKick:
		return true
	case *ws.MessageTaskAnswer:
		return true
	case *ws.MessagePollChoose:
		return true
	default:
		return false
	}
}

func (pollStartedState) name() string {
	return "poll-started"
}

type taskEndedState struct{}

func (taskEndedState) isSessionState() {}

func (taskEndedState) isAllowedMsg(m ws.RecvMessage) bool {
	switch m.(type) {
	case *ws.MessageReady:
		return true
	case *ws.MessageLeave:
		return true
	case *ws.MessageKick:
		return true
	case *ws.MessageTaskAnswer:
		return true
	case *ws.MessagePollChoose:
		return true
	default:
		return false
	}
}

func (taskEndedState) name() string {
	return "task-ended"
}
