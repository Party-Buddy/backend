package session

import "fmt"

// UnsafeStorage stores all the session state.
// By itself it does not provide any concurrency guarantees: if you need them, use a [SyncStorage] instead.
type UnsafeStorage struct {
	sessions    map[SessionId]*session
	inviteCodes map[InviteCode]SessionId
}

func NewUnsafeStorage() UnsafeStorage {
	return UnsafeStorage{
		sessions:    make(map[SessionId]*session),
		inviteCodes: make(map[InviteCode]SessionId),
	}
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
		state: &AwaitingPlayersState{
			InviteCode:   code,
			RequireReady: requireReady,
			Owner:        ownerId,
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

// ForEachPlayer calls f for each player in a session.
func (s *UnsafeStorage) ForEachPlayer(sid SessionId, f func(Player)) {
	session := s.sessions[sid]
	if session == nil {
		return
	}
	for _, player := range session.players {
		f(player)
	}
}

// Players returns all players in a session.
func (s *UnsafeStorage) Players(sid SessionId) (players []Player) {
	s.ForEachPlayer(sid, func(player Player) {
		players = append(players, player)
	})
	return
}

// PlayerTxs returns the Tx fields of each player in a session.
func (s *UnsafeStorage) PlayerTxs(sid SessionId) (txs []TxChan) {
	s.ForEachPlayer(sid, func(player Player) {
		txs = append(txs, player.Tx)
	})
	return
}

// PlayerCount returns the number of players currently in the session.
func (s *UnsafeStorage) PlayerCount(sid SessionId) int {
	if session := s.sessions[sid]; session != nil {
		return len(session.players)
	}

	return 0
}

// SessionFull returns true iff the number of players in a session reached the session's maximum.
func (s *UnsafeStorage) SessionFull(sid SessionId) bool {
	if session := s.sessions[sid]; session != nil {
		return len(session.players) >= session.playersMax
	}
	return false
}

// SessionGame returns the game played in a session.
//
// If the session does not exist, sets ok to false.
func (s *UnsafeStorage) SessionGame(sid SessionId) (game Game, ok bool) {
	if session := s.sessions[sid]; session != nil {
		return session.game, true
	}
	return
}

// SessionState returns a session's current state.
func (s *UnsafeStorage) SessionState(sid SessionId) State {
	if session := s.sessions[sid]; session != nil {
		return session.state
	}
	return nil
}

// HasPlayerNickname returns true iff there's a player with the given nickname in a session.
func (s *UnsafeStorage) HasPlayerNickname(sid SessionId, nickname string) bool {
	session := s.sessions[sid]
	if session == nil {
		return false
	}

	// O(n) in the number of players
	// this is fine: n <= MaxPlayers, which is reasonably small
	for playerId := range session.players {
		if session.players[playerId].Nickname == nickname {
			return true
		}
	}

	return false
}

// ClientBanned checks if a client with the given id is banned from a session.
func (s *UnsafeStorage) ClientBanned(sid SessionId, clientId ClientId) bool {
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

// AddPlayer adds a client to a session as another player.
func (s *UnsafeStorage) AddPlayer(
	sid SessionId,
	clientId ClientId,
	nickname string,
	tx TxChan,
) (player Player, err error) {
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

// AwaitingPlayers returns true iff the current session state is awaitingPlayersState.
func (s *UnsafeStorage) AwaitingPlayers(sid SessionId) bool {
	if session := s.sessions[sid]; session != nil {
		_, ok := session.state.(*AwaitingPlayersState)
		return ok
	}
	return false
}
