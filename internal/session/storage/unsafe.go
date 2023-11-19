package storage

import "fmt"

// UnsafeStorage stores all the session state.
// By itself it does not provide any concurrency guarantees: if you need them, use a [Manager] instead.
type UnsafeStorage struct {
	sessions    map[SessionId]*session
	inviteCodes map[InviteCode]SessionId
}

// InviteCodeLimit is the maximum allowed number of non-expired invite codes.
const InviteCodeLimit = MaxInviteCodeCount / 2

func (s *UnsafeStorage) newInviteCode() (InviteCode, error) {
	if len(s.inviteCodes) >= InviteCodeLimit {
		return InviteCode(""), fmt.Errorf("exceeded the limit on active invite codes: %v", InviteCodeLimit)
	}

	for {
		code := NewInviteCode()
		if _, exists := s.inviteCodes[code]; !exists {
			return code, nil
		}
	}
}

// SessionExists returns `true` if a session with the given `sid` exists.
func (s *UnsafeStorage) SessionExists(sid SessionId) bool {
	return s.sessions[sid] != nil
}

// SidByInviteCode returns the session id of a session with the provided invite code.
// If no such session exists, returns nil.
func (s *UnsafeStorage) SidByInviteCode(code InviteCode) (sid SessionId, ok bool) {
	sid, ok = s.inviteCodes[code]
	return
}

func (s *UnsafeStorage) sessionById(sid SessionId) (*session, error) {
	if session := s.sessions[sid]; session != nil {
		return session, nil
	}

	return nil, fmt.Errorf("session does not exist: %v", sid)
}

// NewSession creates a new session.
//
// The state is set to awaitingPlayersState, and the owner is added as the first player in the lobby.
// Returns the newly created session's ID, its invite code, as well as the [PlayerId] of the owner.
//
// The given game is cloned by NewSession.
func (s *UnsafeStorage) NewSession(
	game *Game,
	owner ClientId,
	ownerNickname string,
	requireReady bool,
	playersMax int,
) (sid SessionId, code InviteCode, ownerId PlayerId, err error) {
	code, err = s.newInviteCode()
	if err != nil {
		return
	}

	sid = NewSessionId()
	ownerId = NewPlayerId()

	s.sessions[sid] = &session{
		id:   sid,
		game: *game,
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
	s.inviteCodes[code] = sid

	return
}

// RemoveSession removes a session from the storage.
func (s *UnsafeStorage) RemoveSession(sid SessionId) {
	// TODO
}

// PlayerByClientId returns a player in a session with the given clientId.
func (s *UnsafeStorage) PlayerByClientId(sid SessionId, clientId ClientId) (Player, error) {
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
func (s *UnsafeStorage) PlayerById(sid SessionId, playerId PlayerId) (player Player, err error) {
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
func (s *UnsafeStorage) PlayerCount(sid SessionId) int {
	if session, ok := s.sessions[sid]; ok {
		return len(session.players)
	}

	return 0
}

// IsClientBanned checks if a client with the given id is banned from a session.
func (s *UnsafeStorage) IsClientBanned(sid SessionId, clientId ClientId) bool {
	session := s.sessions[sid]
	if session == nil {
		return false
	}
	_, ok := session.bannedClients[clientId]
	return ok
}

// banClient adds a client to a list of clients banned from a session.
// The client, if they were present, is not removed from the game.
func (s *UnsafeStorage) banClient(sid SessionId, clientId ClientId) {
	session := s.sessions[sid]
	if session == nil {
		return
	}
	session.bannedClients[clientId] = struct{}{}
}

// addPlayer adds a client to a session as another player.
func (s *UnsafeStorage) addPlayer(sid SessionId, clientId ClientId, nickname string) (player Player, err error) {
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
func (s *UnsafeStorage) removePlayer(sid SessionId, clientId ClientId) (PlayerId, bool) {
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
