package storage

import (
	"fmt"
	"sync"
)

// A Manager encapsulates an `Unsafe` and provides a thread-safe interface to the storage.
type Manager struct {
	mtx   sync.Mutex
	inner Unsafe
}

// Atomically performs the provided operation on the inner storage atomically.
// While the function is being run, no other goroutine may access the inner storage.
// This function is not re-entrant: do not call Atomically in `f`.
func Atomically[R any](mgr *Manager, f func(s *Unsafe) R) R {
	mgr.mtx.Lock()
	defer mgr.mtx.Unlock()

	return f(&mgr.inner)
}

// Unsafe stores all the session state.
// By itself it does not provide any concurrency guarantees: if you need them, use a [Manager] instead.
type Unsafe struct {
	sessions    map[SessionId]*session
	inviteCodes map[InviteCode]SessionId
}

// SessionExists returns `true` if a session with the given `sid` exists.
func (s *Unsafe) SessionExists(sid SessionId) bool {
	return s.sessions[sid] != nil
}

// SidByInviteCode returns the session id of a session with the provided invite code.
// If no such session exists, returns nil.
func (s *Unsafe) SidByInviteCode(code InviteCode) (sid SessionId, ok bool) {
	sid, ok = s.inviteCodes[code]
	return
}

func (s *Unsafe) sessionById(sid SessionId) (*session, error) {
	if session := s.sessions[sid]; session != nil {
		return session, nil
	}

	return nil, fmt.Errorf("session does not exist: %v", sid)
}

// NewSession creates a new session.
// The state is set to awaitingPlayersState, and the owner is added as the first player in the lobby.
// Returns the newly created session's ID, its invite code, as well as the [PlayerId] of the owner.
func (s *Unsafe) NewSession(game Game, owner ClientId, ownerNickname string, requireReady bool, playersMax int) (sid SessionId, code InviteCode, ownerId PlayerId, err error) {
	code = NewInviteCode()
	sid = NewSessionId()
	ownerId = NewPlayerId()

	s.sessions[sid] = &session{
		id:   sid,
		game: game,
		players: map[PlayerId]Player{
			ownerId: {
				Id:       ownerId,
				ClientId: owner,
				Nickname: ownerNickname,
			},
		},
		playersMax: playersMax,
		clients:    map[ClientId]PlayerId{owner: ownerId},
		state: &awaitingPlayersState{
			inviteCode:   code,
			requireReady: requireReady,
			owner:        ownerId,
		},
	}

	return
}

// PlayerByClientId returns a player in a session with the given clientId.
func (s *Unsafe) PlayerByClientId(sid SessionId, clientId ClientId) (Player, error) {
	session, err := s.sessionById(sid)
	if err != nil {
		return Player{}, err
	}

	playerId, ok := session.clients[clientId]
	if !ok {
		return Player{}, fmt.Errorf("client is not a player: %v", clientId)
	}

	return session.players[playerId], nil
}

// PlayerById returns a player in a session with the given playerId.
func (s *Unsafe) PlayerById(sid SessionId, playerId PlayerId) (player Player, err error) {
	session, err := s.sessionById(sid)
	if err != nil {
		return
	}

	player, ok := session.players[playerId]
	if !ok {
		return player, fmt.Errorf("invalid player id for session %v: %v", sid, playerId)
	}

	return
}

// PlayerCount returns the number of players currently in the session.
func (s *Unsafe) PlayerCount(sid SessionId) int {
	if session, ok := s.sessions[sid]; ok {
		return len(session.players)
	}

	return 0
}

// IsClientBanned checks if a client with the given id is banned from a session.
func (s *Unsafe) IsClientBanned(sid SessionId, clientId ClientId) bool {
	session := s.sessions[sid]
	if session == nil {
		return false
	}
	_, ok := session.bannedClients[clientId]
	return ok
}

// banClient adds a client to a list of clients banned from a session.
// The client, if they were present, is not removed from the game.
func (s *Unsafe) banClient(sid SessionId, clientId ClientId) {
	session := s.sessions[sid]
	if session == nil {
		return
	}
	session.bannedClients[clientId] = struct{}{}
}

// addPlayer adds a client to a session as another player.
func (s *Unsafe) addPlayer(sid SessionId, clientId ClientId, nickname string) (player Player, err error) {
	session, err := s.sessionById(sid)
	if err != nil {
		return
	}

	if playerId, ok := session.clients[clientId]; ok {
		player = session.players[playerId]
		return player, fmt.Errorf("client %v is already a player: %+v", clientId, player)
	}

	playerId := NewPlayerId()
	player = Player{
		Id:       playerId,
		ClientId: clientId,
		Nickname: nickname,
	}
	session.players[playerId] = player
	session.clients[clientId] = playerId

	return
}

// removePlayer removes a client from a session.
func (s *Unsafe) removePlayer(sid SessionId, clientId ClientId) (PlayerId, bool) {
	session := s.sessions[sid]
	if session == nil {
		return PlayerId{}, false
	}

	playerId, ok := session.clients[clientId]
	if !ok {
		return PlayerId{}, false
	}

	delete(session.players, playerId)
	delete(session.clients, clientId)

	return playerId, true
}
