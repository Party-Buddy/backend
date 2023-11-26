package session

import (
	"fmt"
	"sync"
)

// A SyncStorage encapsulates an [UnsafeStorage] and provides a thread-safe interface to the storage.
type SyncStorage struct {
	mtx   sync.Mutex
	inner UnsafeStorage
}

func NewSyncStorage() SyncStorage {
	return SyncStorage{
		mtx:   sync.Mutex{},
		inner: NewUnsafeStorage(),
	}
}

// Atomically performs the provided operation on the inner storage atomically.
// While the function is being run, no other goroutine may access the inner storage.
// This function is not re-entrant: do not call Atomically in `f`.
func (s *SyncStorage) Atomically(f func(s *UnsafeStorage)) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	f(&s.inner)
}

// UnsafeStorage stores all the session state.
// By itself it does not provide any concurrency guarantees: if you need them, use a [SyncStorage] instead.
type UnsafeStorage struct {
	sessions    map[SessionID]*session
	inviteCodes map[InviteCode]SessionID
}

func NewUnsafeStorage() UnsafeStorage {
	return UnsafeStorage{
		sessions:    make(map[SessionID]*session),
		inviteCodes: make(map[InviteCode]SessionID),
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
func (s *UnsafeStorage) SessionExists(sid SessionID) bool {
	return s.sessions[sid] != nil
}

// SidByInviteCode returns the session id of a session with the provided invite code.
// If no such session exists, returns nil.
func (s *UnsafeStorage) SidByInviteCode(code InviteCode) (sid SessionID, ok bool) {
	sid, ok = s.inviteCodes[code]
	return
}

func (s *UnsafeStorage) sessionByID(sid SessionID) (*session, error) {
	if session := s.sessions[sid]; session != nil {
		return session, nil
	}

	return nil, fmt.Errorf("session does not exist: %v", sid)
}

// newSession creates a new session.
//
// The state is set to awaitingPlayersState.
// The owner is **not** added to the session.
// Returns the newly created session's ID and its invite code.
//
// The given game is cloned by newSession.
func (s *UnsafeStorage) newSession(
	game *Game,
	ownerID ClientID,
	requireReady bool,
	playersMax int,
) (sid SessionID, code InviteCode, err error) {
	code, err = s.newInviteCode()
	if err != nil {
		return
	}

	sid = NewSessionID()

	s.sessions[sid] = &session{
		id:            sid,
		game:          *game,
		players:       make(map[PlayerID]Player),
		playersMax:    playersMax,
		clients:       make(map[ClientID]PlayerID),
		bannedClients: make(map[ClientID]struct{}),
		state: &AwaitingPlayersState{
			inviteCode:   code,
			requireReady: requireReady,
			owner:        ownerID,
		},
	}
	s.inviteCodes[code] = sid

	return
}

// removeSession removes a session from the storage.
func (s *UnsafeStorage) removeSession(sid SessionID) {
	// TODO
}

// PlayerByClientID returns a player in a session with the given clientID.
func (s *UnsafeStorage) PlayerByClientID(sid SessionID, clientID ClientID) (Player, error) {
	session, err := s.sessionByID(sid)
	if err != nil {
		return Player{}, err
	}

	playerID, ok := session.clients[clientID]
	if !ok {
		return Player{}, fmt.Errorf("client is not a player: %v", clientID)
	}

	return session.players[playerID], nil
}

// PlayerByID returns a player in a session with the given playerID.
func (s *UnsafeStorage) PlayerByID(sid SessionID, playerID PlayerID) (player Player, err error) {
	session, err := s.sessionByID(sid)
	if err != nil {
		return
	}

	player, ok := session.players[playerID]
	if !ok {
		return player, fmt.Errorf("invalid player id for session %v: %v", sid, playerID)
	}

	return
}

// ForEachPlayer calls f for each player in a session.
func (s *UnsafeStorage) ForEachPlayer(sid SessionID, f func(Player)) {
	session := s.sessions[sid]
	if session == nil {
		return
	}
	for _, player := range session.players {
		f(player)
	}
}

// Players returns all players in a session.
func (s *UnsafeStorage) Players(sid SessionID) (players []Player) {
	s.ForEachPlayer(sid, func(player Player) {
		players = append(players, player)
	})
	return
}

// PlayerTxs returns a Tx channel for each player in a session.
func (s *UnsafeStorage) PlayerTxs(sid SessionID) (txs []TxChan) {
	s.ForEachPlayer(sid, func(player Player) {
		txs = append(txs, player.Tx)
	})
	return
}

// PlayerCount returns the number of players currently in the session.
func (s *UnsafeStorage) PlayerCount(sid SessionID) int {
	if session := s.sessions[sid]; session != nil {
		return len(session.players)
	}

	return 0
}

// SessionFull returns true iff the number of players in a session reached the session's maximum.
func (s *UnsafeStorage) SessionFull(sid SessionID) bool {
	if session := s.sessions[sid]; session != nil {
		return len(session.players) >= session.playersMax
	}
	return false
}

// SessionGame returns the game played in a session.
//
// If the session does not exist, sets ok to false.
func (s *UnsafeStorage) SessionGame(sid SessionID) (game Game, ok bool) {
	if session := s.sessions[sid]; session != nil {
		return session.game, true
	}
	return
}

// SessionState returns a session's current state.
func (s *UnsafeStorage) sessionState(sid SessionID) State {
	if session := s.sessions[sid]; session != nil {
		return session.state
	}
	return nil
}

// HasPlayerNickname returns true iff there's a player with the given nickname in a session.
func (s *UnsafeStorage) HasPlayerNickname(sid SessionID, nickname string) bool {
	session := s.sessions[sid]
	if session == nil {
		return false
	}

	// O(n) in the number of players
	// this is fine: n <= MaxPlayers, which is reasonably small
	for playerID := range session.players {
		if session.players[playerID].Nickname == nickname {
			return true
		}
	}

	return false
}

// ClientBanned checks if a client with the given id is banned from a session.
func (s *UnsafeStorage) ClientBanned(sid SessionID, clientID ClientID) bool {
	session := s.sessions[sid]
	if session == nil {
		return false
	}
	_, ok := session.bannedClients[clientID]
	return ok
}

// banClient adds a client to a list of clients banned from a session.
// The client, if they were present, is not removed from the game.
func (s *UnsafeStorage) banClient(sid SessionID, clientID ClientID) {
	session := s.sessions[sid]
	if session == nil {
		return
	}
	session.bannedClients[clientID] = struct{}{}
}

// AddPlayer adds a client to a session as another player.
func (s *UnsafeStorage) addPlayer(
	sid SessionID,
	clientID ClientID,
	nickname string,
	tx TxChan,
) (player Player, err error) {
	session, err := s.sessionByID(sid)
	if err != nil {
		return
	}

	if playerID, ok := session.clients[clientID]; ok {
		player = session.players[playerID]
		return player, fmt.Errorf("client %v is already a player: %+v", clientID, player)
	}

	playerID := NewPlayerID()
	player = Player{
		ID:       playerID,
		ClientID: clientID,
		Nickname: nickname,
	}
	session.players[playerID] = player
	session.clients[clientID] = playerID

	return
}

// removePlayer removes a client from a session.
func (s *UnsafeStorage) removePlayer(sid SessionID, clientID ClientID) (PlayerID, bool) {
	session := s.sessions[sid]
	if session == nil {
		return PlayerID{}, false
	}

	playerID, ok := session.clients[clientID]
	if !ok {
		return PlayerID{}, false
	}

	delete(session.players, playerID)
	delete(session.clients, clientID)

	return playerID, true
}

// AwaitingPlayers returns true iff the current session state is awaitingPlayersState.
func (s *UnsafeStorage) AwaitingPlayers(sid SessionID) bool {
	if session := s.sessions[sid]; session != nil {
		_, ok := session.state.(*AwaitingPlayersState)
		return ok
	}
	return false
}
