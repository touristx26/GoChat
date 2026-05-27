package server

import (
	"log"
	"sync"
	"time"

	"github.com/xianfeng-wang/gochat/internal/protocol"
)

type Hub struct {
	clients    map[string]*Client
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	h.getOrCreateRoom("general")
	return h
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			h.sendWelcome(client)
			log.Printf("client connected: %s", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			_, ok := h.clients[client.ID]
			if ok {
				h.removeClientFromAllRooms(client)
				delete(h.clients, client.ID)
				close(client.send)
			}
			h.mu.Unlock()
			if ok {
				h.broadcastRoomList()
			}
			log.Printf("client disconnected: %s (%s)", client.ID, client.Username)
		}
	}
}

func (h *Hub) handleMessage(client *Client, data []byte) {
	msg, err := protocol.Decode(data)
	if err != nil {
		h.sendError(client, "invalid_message", "invalid message format")
		return
	}

	switch msg.Type {
	case protocol.TypeSetUsername:
		h.handleSetUsername(client, msg.Payload)
	case protocol.TypeJoinRoom:
		h.handleJoinRoom(client, msg.Payload)
	case protocol.TypeLeaveRoom:
		h.handleLeaveRoom(client, msg.Payload)
	case protocol.TypeSendMessage:
		h.handleSendMessage(client, msg.Payload)
	case protocol.TypeListRooms:
		h.handleListRooms(client)
	case protocol.TypeListUsers:
		h.handleListUsers(client, msg.Payload)
	default:
		h.sendError(client, "unknown_type", "unknown message type: "+msg.Type)
	}
}

func (h *Hub) handleSetUsername(client *Client, raw []byte) {
	payload, err := protocol.DecodePayload[protocol.SetUsernamePayload](raw)
	if err != nil || payload.Username == "" {
		h.sendError(client, "invalid_payload", "username is required")
		return
	}
	oldName := client.Username
	client.Username = payload.Username
	log.Printf("client %s set username: %s -> %s", client.ID, oldName, payload.Username)

	for roomName := range client.rooms {
		h.mu.RLock()
		room, exists := h.rooms[roomName]
		h.mu.RUnlock()
		if !exists {
			continue
		}
		notify, _ := protocol.Encode(protocol.TypeUsernameChanged, protocol.UsernameChangedPayload{
			Room:    roomName,
			OldName: oldName,
			NewName: payload.Username,
		})
		room.Broadcast(notify, nil)
	}
}

func (h *Hub) handleJoinRoom(client *Client, raw []byte) {
	payload, err := protocol.DecodePayload[protocol.JoinRoomPayload](raw)
	if err != nil || payload.Room == "" {
		h.sendError(client, "invalid_payload", "room name is required")
		return
	}
	if client.Username == "" {
		h.sendError(client, "no_username", "set username before joining a room")
		return
	}

	h.mu.Lock()
	room := h.getOrCreateRoom(payload.Room)
	h.mu.Unlock()

	if client.rooms[payload.Room] {
		h.sendError(client, "already_joined", "already in room: "+payload.Room)
		return
	}

	room.Join(client)
	client.rooms[payload.Room] = true

	joined, _ := protocol.Encode(protocol.TypeRoomJoined, protocol.RoomJoinedPayload{
		Room:  payload.Room,
		Users: room.Usernames(),
	})
	client.Send(joined)

	notify, _ := protocol.Encode(protocol.TypeUserJoined, protocol.UserJoinedPayload{
		Room:     payload.Room,
		Username: client.Username,
	})
	room.Broadcast(notify, client)
	h.broadcastRoomList()
}

func (h *Hub) handleLeaveRoom(client *Client, raw []byte) {
	payload, err := protocol.DecodePayload[protocol.LeaveRoomPayload](raw)
	if err != nil || payload.Room == "" {
		h.sendError(client, "invalid_payload", "room name is required")
		return
	}

	if !client.rooms[payload.Room] {
		h.sendError(client, "not_in_room", "not in room: "+payload.Room)
		return
	}

	h.mu.RLock()
	room, exists := h.rooms[payload.Room]
	h.mu.RUnlock()
	if !exists {
		return
	}

	room.Leave(client)
	delete(client.rooms, payload.Room)

	left, _ := protocol.Encode(protocol.TypeRoomLeft, protocol.RoomLeftPayload{Room: payload.Room})
	client.Send(left)

	notify, _ := protocol.Encode(protocol.TypeUserLeft, protocol.UserLeftPayload{
		Room:     payload.Room,
		Username: client.Username,
	})
	room.Broadcast(notify, nil)
	h.broadcastRoomList()
}

func (h *Hub) handleSendMessage(client *Client, raw []byte) {
	payload, err := protocol.DecodePayload[protocol.SendMessagePayload](raw)
	if err != nil || payload.Room == "" || payload.Content == "" {
		h.sendError(client, "invalid_payload", "room and content are required")
		return
	}

	if !client.rooms[payload.Room] {
		h.sendError(client, "not_in_room", "not in room: "+payload.Room)
		return
	}

	h.mu.RLock()
	room, exists := h.rooms[payload.Room]
	h.mu.RUnlock()
	if !exists {
		return
	}

	chatMsg, _ := protocol.Encode(protocol.TypeChatMessage, protocol.ChatMessagePayload{
		Room:      payload.Room,
		From:      client.Username,
		Content:   payload.Content,
		Timestamp: time.Now().Format("15:04:05"),
	})
	room.Broadcast(chatMsg, nil)
}

func (h *Hub) handleListRooms(client *Client) {
	h.mu.RLock()
	rooms := make([]protocol.RoomInfo, 0, len(h.rooms))
	for _, r := range h.rooms {
		rooms = append(rooms, protocol.RoomInfo{
			Name:      r.Name,
			UserCount: r.UserCount(),
		})
	}
	h.mu.RUnlock()

	data, _ := protocol.Encode(protocol.TypeRoomList, protocol.RoomListPayload{Rooms: rooms})
	client.Send(data)
}

func (h *Hub) handleListUsers(client *Client, raw []byte) {
	payload, err := protocol.DecodePayload[protocol.ListUsersPayload](raw)
	if err != nil || payload.Room == "" {
		h.sendError(client, "invalid_payload", "room name is required")
		return
	}

	h.mu.RLock()
	room, exists := h.rooms[payload.Room]
	h.mu.RUnlock()
	if !exists {
		h.sendError(client, "room_not_found", "room not found: "+payload.Room)
		return
	}

	data, _ := protocol.Encode(protocol.TypeUserList, protocol.UserListPayload{
		Room:  payload.Room,
		Users: room.Usernames(),
	})
	client.Send(data)
}

func (h *Hub) sendWelcome(client *Client) {
	h.mu.RLock()
	rooms := make([]protocol.RoomInfo, 0, len(h.rooms))
	for _, r := range h.rooms {
		rooms = append(rooms, protocol.RoomInfo{
			Name:      r.Name,
			UserCount: r.UserCount(),
		})
	}
	h.mu.RUnlock()

	data, _ := protocol.Encode(protocol.TypeWelcome, protocol.WelcomePayload{
		UserID: client.ID,
		Rooms:  rooms,
	})
	client.Send(data)
}

func (h *Hub) sendError(client *Client, code, message string) {
	data, _ := protocol.Encode(protocol.TypeError, protocol.ErrorPayload{
		Code:    code,
		Message: message,
	})
	client.Send(data)
}

func (h *Hub) getOrCreateRoom(name string) *Room {
	if room, ok := h.rooms[name]; ok {
		return room
	}
	room := NewRoom(name)
	h.rooms[name] = room
	return room
}

func (h *Hub) removeClientFromAllRooms(client *Client) {
	for roomName := range client.rooms {
		if room, ok := h.rooms[roomName]; ok {
			room.Leave(client)
			notify, _ := protocol.Encode(protocol.TypeUserLeft, protocol.UserLeftPayload{
				Room:     roomName,
				Username: client.Username,
			})
			room.Broadcast(notify, nil)
		}
	}
}

func (h *Hub) broadcastRoomList() {
	h.mu.RLock()
	rooms := make([]protocol.RoomInfo, 0, len(h.rooms))
	for _, r := range h.rooms {
		rooms = append(rooms, protocol.RoomInfo{
			Name:      r.Name,
			UserCount: r.UserCount(),
		})
	}
	clients := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	data, _ := protocol.Encode(protocol.TypeRoomList, protocol.RoomListPayload{Rooms: rooms})
	for _, c := range clients {
		c.Send(data)
	}
}
