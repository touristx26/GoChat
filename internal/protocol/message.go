package protocol

import "encoding/json"

const (
	// Client -> Server
	TypeSetUsername = "set_username"
	TypeJoinRoom   = "join_room"
	TypeLeaveRoom  = "leave_room"
	TypeSendMessage = "send_message"
	TypeListRooms  = "list_rooms"
	TypeListUsers  = "list_users"

	// Server -> Client
	TypeWelcome       = "welcome"
	TypeError         = "error"
	TypeRoomJoined    = "room_joined"
	TypeRoomLeft      = "room_left"
	TypeChatMessage   = "chat_message"
	TypeUserJoined    = "user_joined"
	TypeUserLeft      = "user_left"
	TypeRoomList      = "room_list"
	TypeUserList      = "user_list"
	TypeSystemMessage = "system_message"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func Encode(msgType string, payload any) ([]byte, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Message{Type: msgType, Payload: p})
}

func Decode(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}

func DecodePayload[T any](raw json.RawMessage) (T, error) {
	var payload T
	err := json.Unmarshal(raw, &payload)
	return payload, err
}
