package ws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"party-buddy/internal/schemas/ws"
	"party-buddy/internal/session"
	"party-buddy/internal/validate"
	"party-buddy/internal/ws/converters"
	"party-buddy/internal/ws/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cohesivestack/valgo"
	"github.com/gorilla/websocket"
)

type Conn struct {
	manager *session.Manager

	// wsConn is a WebSocket (ws) connection
	wsConn *websocket.Conn

	// client is the ClientID to which ws connection is related
	client session.ClientID

	// sid is the SessionID to which ws connection is related
	sid session.SessionID

	// msgToClientChan is the channel for messages ready to send to client
	msgToClientChan chan<- ws.RespMessage

	// playerID is the player identifier inside the game
	playerID *session.PlayerID

	// msgID is used for getting new msg-id
	// DO NOT get the data by accessing field
	// use nextMsgID instead
	msgID atomic.Uint32

	// stopRequested indicates that wsConn and channels should be closed
	stopRequested atomic.Bool

	// cancel is a function to call for cancelling runWriter, runServeToWriterConverter
	cancel context.CancelFunc

	// servDataChan here for closing
	servDataChan session.TxChan

	mainLog   *log.Logger
	readerLog *log.Logger
	writerLog *log.Logger
	serverLog *log.Logger

	// state is the current web socket protocol state
	state sessionState

	// stateMtx should be locked before accessing the state
	stateMtx sync.Mutex
}

func logPrefix(sid session.SessionID, clientID session.ClientID, playerID *session.PlayerID, sub string) string {
	var prefix strings.Builder

	fmt.Fprintf(&prefix, "ws.Conn(sid %s, client %s", sid, clientID)
	if playerID != nil {
		fmt.Fprintf(&prefix, ", player %s", *playerID)
	}
	prefix.WriteString("): ")

	if sub != "" {
		fmt.Fprintf(&prefix, "%s: ", sub)
	}

	return prefix.String()
}

func NewConn(
	parentLogger *log.Logger,
	manager *session.Manager,
	wsConn *websocket.Conn,
	clientID session.ClientID,
	sid session.SessionID,
) *Conn {
	mainLog := log.New(parentLogger.Writer(), logPrefix(sid, clientID, nil, ""), parentLogger.Flags())
	readerLog := log.New(mainLog.Writer(), logPrefix(sid, clientID, nil, "reader"), mainLog.Flags())
	writerLog := log.New(mainLog.Writer(), logPrefix(sid, clientID, nil, "writer"), mainLog.Flags())
	serverLog := log.New(mainLog.Writer(), logPrefix(sid, clientID, nil, "mgr-rx"), mainLog.Flags())

	return &Conn{
		manager:       manager,
		wsConn:        wsConn,
		client:        clientID,
		sid:           sid,
		msgID:         atomic.Uint32{},
		stopRequested: atomic.Bool{},
		mainLog:       mainLog,
		readerLog:     readerLog,
		writerLog:     writerLog,
		serverLog:     serverLog,
	}
}

func (c *Conn) StartReadAndWriteConn(f *valgo.ValidationFactory) {
	c.stopRequested.Store(false)
	servChan := make(chan session.ServerTx)
	msgChan := make(chan ws.RespMessage)
	c.msgToClientChan = msgChan
	ctx, cancel := context.WithCancel(context.Background())
	ctx = validate.NewContext(ctx, f)
	c.cancel = cancel
	c.state = initialState{}
	c.stateMtx = sync.Mutex{}
	go c.runReader(ctx, servChan)
	go c.runServeToWriterConverter(ctx, msgChan, servChan)
	go c.runWriter(ctx, msgChan)
	c.mainLog.Println("started")
}

func (c *Conn) setPlayerID(playerID session.PlayerID) {
	c.playerID = &playerID
	c.mainLog.SetPrefix(logPrefix(c.sid, c.client, c.playerID, ""))
	c.readerLog.SetPrefix(logPrefix(c.sid, c.client, c.playerID, "reader"))
	c.writerLog.SetPrefix(logPrefix(c.sid, c.client, c.playerID, "writer"))
	c.serverLog.SetPrefix(logPrefix(c.sid, c.client, c.playerID, "mgr-rx"))
}

func (c *Conn) runServeToWriterConverter(
	ctx context.Context,
	msgChan chan<- ws.RespMessage,
	servChan <-chan session.ServerTx,
) {
	defer func() {
		c.serverLog.Println("closing the writer channel")
		close(msgChan)
		c.serverLog.Println("stopping")
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-servChan:
			if msg == nil {
				c.serverLog.Println("the server channel has been closed, stopping")
				c.stopRequested.Store(true)
				return
			}

			c.serverLog.Printf("handling %T received via the server channel", msg)
			if c.stopRequested.Load() {
				c.serverLog.Printf("stop requested, skipping message handling")
				continue
			}

			var clientMessage ws.RespMessage
			switch m := msg.(type) {
			case *session.MsgError:
				refID := msgIDFromContext(m.Context())
				kind, errMsg := converters.ErrorCodeAndMessage(m.Inner)
				errorMsg := utils.GenMessageError(refID, kind, errMsg)
				clientMessage = &errorMsg

			case *session.MsgJoined:
				joinedMsg := converters.ToMessageJoined(*m)
				joinedMsg.RefID = msgIDFromContext(m.Context())
				clientMessage = &joinedMsg

				c.stateMtx.Lock()
				c.state = awaitingPlayersState{}
				c.stateMtx.Unlock()

			case *session.MsgGameStatus:
				gameStatusMsg := converters.ToMessageGameStatus(*m)
				clientMessage = &gameStatusMsg

			case *session.MsgTaskStart:
				taskStartMsg := converters.ToMessageTaskStart(*m)
				clientMessage = &taskStartMsg

				c.stateMtx.Lock()
				c.state = taskStartedState{}
				c.stateMtx.Unlock()

			case *session.MsgTaskEnd:
				taskEndMsg := converters.ToMessageTaskEnd(*m)
				clientMessage = &taskEndMsg

				c.stateMtx.Lock()
				c.state = taskEndedState{}
				c.stateMtx.Unlock()

			case *session.MsgGameStart:
				gameStartMsg := converters.ToMessageGameStart(*m)
				clientMessage = &gameStartMsg

				c.stateMtx.Lock()
				c.state = gameStartedState{}
				c.stateMtx.Unlock()

			case *session.MsgWaiting:
				waitingMsg := converters.ToMessageWaiting(*m)
				clientMessage = &waitingMsg

				c.stateMtx.Lock()
				c.state = awaitingPlayersState{}
				c.stateMtx.Unlock()

			case *session.MsgGameEnd:
				gameEndMsg := converters.ToMessageGameEnd(*m)
				clientMessage = &gameEndMsg
			}

			if clientMessage == nil {
				c.serverLog.Println("unknown msg from server")
				continue
			}
			msgChan <- clientMessage
		}
	}
}

func (c *Conn) runWriter(ctx context.Context, msgChan <-chan ws.RespMessage) {
	defer func() {
		c.writerLog.Println("stopping")
		properWSClose(c.wsConn)
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-msgChan:
			if msg == nil {
				c.writerLog.Println("the writer channel has been closed, canceling the context")
				return
			}

			msgID := ws.MessageID(c.nextMsgID())
			msg.SetMsgID(msgID)

			c.writerLog.Printf("sending `%s` to the client (msg-id %d)", msg.GetKind(), msgID)
			err := c.wsConn.WriteJSON(msg)

			if err != nil {
				c.writerLog.Printf("encountered an error while sending a message: %s", err)
			}
		}
	}
}

type msgIDKeyType int

var msgIDKey msgIDKeyType

func msgIDFromContext(ctx context.Context) *ws.MessageID {
	if msgID, ok := ctx.Value(msgIDKey).(ws.MessageID); ok {
		return &msgID
	}
	return nil
}

func (c *Conn) runReader(ctx context.Context, servDataChan session.TxChan) {
	defer c.readerLog.Printf("stopping")
	for !c.stopRequested.Load() {
		_, bytes, err := c.wsConn.ReadMessage()
		if err != nil {
			c.readerLog.Printf("ReadMessage failed: %s", err)

			if !c.stopRequested.Load() {
				c.dispose(ctx)
			}
			return
		}
		msg, err := ws.ParseMessage(ctx, bytes)
		if err != nil {
			var errDto *ws.Error
			errors.As(ws.ParseErrorToMessageError(err), &errDto)
			rspMessage := ws.MessageError{
				BaseMessage: utils.GenBaseMessage(&ws.MsgKindError),
				Error:       *errDto,
			}
			c.readerLog.Printf("ParseMessage failed: %s (code `%s`)", err, errDto.Code)
			c.msgToClientChan <- &rspMessage

			c.dispose(ctx)
			return
		}

		c.stateMtx.Lock()
		st := c.state
		c.stateMtx.Unlock()
		if !st.isAllowedMsg(msg) {
			id := msg.GetMsgID()
			errMsg := utils.GenMessageError(&id, ws.ErrProtoViolation,
				fmt.Sprintf("the message `%s` is not allowed in the current state", msg.GetKind()))
			c.readerLog.Printf("received a message `%s`: not allowed in the current state `%s` (error code `%v`)",
				msg.GetKind(), st.name(), errMsg.Code)
			c.msgToClientChan <- &errMsg

			c.dispose(ctx)
			return
		}

		ctx = context.WithValue(ctx, msgIDKey, msg.GetMsgID())

		switch m := msg.(type) {
		case *ws.MessageJoin:
			c.readerLog.Println("handling message Join")
			c.handleJoin(ctx, m, servDataChan)

		case *ws.MessageReady:
			c.readerLog.Println("handling message Ready")
			c.handleReady(ctx, m)

		case *ws.MessageTaskAnswer:
			c.readerLog.Println("handling message TaskAnswer")
			c.handleTaskAnswer(ctx, m)
		}
	}
}

func properWSClose(wsConn *websocket.Conn) {
	timeout := 10 * time.Second
	_ = wsConn.SetWriteDeadline(time.Now().Add(timeout))
	_ = wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(timeout)
	_ = wsConn.Close()
}

// dispose is used for closing ws connection and related channels.
// There 2 possible cases to call dispose:
//  1. reader call dispose and the client had NOT joined the session (so it has no PlayerID)
//  2. reader call dispose and client had joined the session
//
// Disconnecting because of server initiative is handled in runServeToWriterConverter
func (c *Conn) dispose(ctx context.Context) {
	c.mainLog.Println("disconnecting")
	c.stopRequested.Store(true)
	if c.playerID != nil { // playerID indicates that client has already joined
		// Here we are asking manager to disconnect us
		c.mainLog.Printf("removing the player from the session")
		c.manager.RemovePlayer(ctx, c.sid, *c.playerID)
	} else {
		// Manager knows nothing about client, so we just stop threads
		c.mainLog.Printf("canceling the context")
		c.cancel()
	}
}

func (c *Conn) nextMsgID() uint32 {
	return c.msgID.Add(1)
}
