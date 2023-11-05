package storage

import "sync"

// A thread-safe storage for session state.
type SessionManager struct {
	mtx         sync.Mutex
	sessions    map[SessionId]*session
	inviteCodes map[InviteCode]SessionId
}

// TODO: now the challenge is figuring out how to approach the actual thread-safety that we're claiming here.
// The options I'm envisioning at the moment are as follows:
//
//  1. Have every method lock a mutex, fetch the value, and unlock the mutex.
//     If we want to fetch multiple things, we do multiple calls.
//     (Not a good idea if you need them to be consistent with each other, true, but you can just throw more methods at a problem.)
//     As long as the mutex is not too contended for, the locking itself shouldn't be an issue.
//     What is an issue is that the mutex must be recursive, and `sync.Mutex` is not.
//     And it's a general belief that recursive mutexes are abhorrent anyway, so let's just forget about this, all right?
//
//  2. Another option would be having two sets of methods, namely Sync and Unsync.
//     One would do the lock/unlock dance, whereas the other would assume the lock is held for the duration of the call and just grab the needed value.
//     While this is more flexible, it requires a ton of boilerplate.
//     Moreover, exporting unsync methods is only useful if we also expose the mutex.
//
//  3. At this point we can just expose the mutex right away and tell everybody they have to lock it before accessing any methods.
//     This is the most flexible as well as the most dangerous option (short of not locking at all, of course).
//     Seeing as Go does not exactly shy away from danger, we should probably go with this one?
//     I'm sure reluctant to commit to this choice...
//     Also, there's absolutely no thread-safety provided by the SessionManager here.
//
//  4. "But wait, it's Go, just use channels!"
//     So what this boils down to is, make a RPC solution built on top of Go channels.
//     You'd have one channel accept ingress data — messages telling what things to get (or update).
//     Discerning the intent could be done by matching on the message type, for instance.
//     Each message would also contain a one-shot channel for sending the response.
//     (Like self-addressed stamped envelopes.)
//     A goroutine would then read incoming messages, do whatever was requested, and send the reply back to the provided channel.
//     What do we gain?
//     A huge increase in complexity and decrease in robustness, for one.
//     A lot of overhead because this essentially re-invents method calls here.
//     Except all calls are indirect and require casts.
//     And messages are heap-allocated.
//     Enough to say, this is too cursed to consider implementing.
//
//  5. There also was an idea involving accepting a callback (lock, call the callback, unlock, return).
//     But I've decided to scratch it.
//     One complaint is that it not the "Go way", whatever cursed abomination this stands for.
//     A more sensible objection is that it's just a bit overcomplicated.
//     In my defense, it makes the extent of a critical session more clear, and you can't forget to unlock the mutex this way.
//
// Well, ugh.
// Let me dwell on this for a bit longer.

func (m *SessionManager) Mutex() *sync.Mutex {
	return &m.mtx
}

// Returns the number of players in the session.
// You must lock the mutex first before accessing this method.
func (m *SessionManager) UnsyncPlayerCount(sid SessionId) int {
	return len(m.sessions[sid].players)
}
