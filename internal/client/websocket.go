package client

import (
	"encoding/json"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/xianfeng-wang/gochat/internal/protocol"
)

type WSClient struct {
	conn     *websocket.Conn
	addr     string
	msgChan  chan tea.Msg
	doneChan chan struct{}
}

func NewWSClient(addr string) *WSClient {
	return &WSClient{
		addr:     addr,
		msgChan:  make(chan tea.Msg, 64),
		doneChan: make(chan struct{}),
	}
}

func (ws *WSClient) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(ws.addr, nil)
	if err != nil {
		return err
	}
	ws.conn = conn
	go ws.readLoop()
	return nil
}

func (ws *WSClient) Close() {
	if ws.conn != nil {
		ws.conn.Close()
	}
}

func (ws *WSClient) Send(msgType string, payload any) error {
	data, err := protocol.Encode(msgType, payload)
	if err != nil {
		return err
	}
	return ws.conn.WriteMessage(websocket.TextMessage, data)
}

func (ws *WSClient) ListenCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-ws.msgChan:
			return msg
		case <-ws.doneChan:
			return DisconnectedMsg{}
		}
	}
}

func (ws *WSClient) readLoop() {
	defer close(ws.doneChan)
	for {
		_, data, err := ws.conn.ReadMessage()
		if err != nil {
			ws.msgChan <- DisconnectedMsg{Err: err}
			return
		}
		msg := ws.parseMessage(data)
		if msg != nil {
			ws.msgChan <- msg
		}
	}
}

func (ws *WSClient) parseMessage(data []byte) tea.Msg {
	var envelope protocol.Message
	if err := json.Unmarshal(data, &envelope); err != nil {
		log.Printf("parse error: %v", err)
		return nil
	}

	switch envelope.Type {
	case protocol.TypeWelcome:
		p, _ := protocol.DecodePayload[protocol.WelcomePayload](envelope.Payload)
		rooms := make([]RoomEntry, len(p.Rooms))
		for i, r := range p.Rooms {
			rooms[i] = RoomEntry{Name: r.Name, UserCount: r.UserCount}
		}
		return ConnectedMsg{UserID: p.UserID}

	case protocol.TypeChatMessage:
		p, _ := protocol.DecodePayload[protocol.ChatMessagePayload](envelope.Payload)
		return ChatMsg{Room: p.Room, From: p.From, Content: p.Content, Timestamp: p.Timestamp}

	case protocol.TypeRoomJoined:
		p, _ := protocol.DecodePayload[protocol.RoomJoinedPayload](envelope.Payload)
		return RoomJoinedMsg{Room: p.Room, Users: p.Users}

	case protocol.TypeRoomLeft:
		p, _ := protocol.DecodePayload[protocol.RoomLeftPayload](envelope.Payload)
		return RoomLeftMsg{Room: p.Room}

	case protocol.TypeUserJoined:
		p, _ := protocol.DecodePayload[protocol.UserJoinedPayload](envelope.Payload)
		return UserJoinedMsg{Room: p.Room, Username: p.Username}

	case protocol.TypeUserLeft:
		p, _ := protocol.DecodePayload[protocol.UserLeftPayload](envelope.Payload)
		return UserLeftMsg{Room: p.Room, Username: p.Username}

	case protocol.TypeRoomList:
		p, _ := protocol.DecodePayload[protocol.RoomListPayload](envelope.Payload)
		rooms := make([]RoomEntry, len(p.Rooms))
		for i, r := range p.Rooms {
			rooms[i] = RoomEntry{Name: r.Name, UserCount: r.UserCount}
		}
		return RoomListMsg{Rooms: rooms}

	case protocol.TypeUserList:
		p, _ := protocol.DecodePayload[protocol.UserListPayload](envelope.Payload)
		return UserListMsg{Room: p.Room, Users: p.Users}

	case protocol.TypeError:
		p, _ := protocol.DecodePayload[protocol.ErrorPayload](envelope.Payload)
		return ErrorMsg{Code: p.Code, Message: p.Message}

	case protocol.TypeSystemMessage:
		p, _ := protocol.DecodePayload[protocol.SystemMessagePayload](envelope.Payload)
		return SystemMsg{Room: p.Room, Content: p.Content}
	}

	return nil
}
