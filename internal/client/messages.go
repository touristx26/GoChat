package client

import tea "github.com/charmbracelet/bubbletea"

type ConnectedMsg struct {
	UserID string
}

type DisconnectedMsg struct {
	Err error
}

type ChatMsg struct {
	Room      string
	From      string
	Content   string
	Timestamp string
}

type SystemMsg struct {
	Room    string
	Content string
}

type RoomJoinedMsg struct {
	Room  string
	Users []string
}

type RoomLeftMsg struct {
	Room string
}

type UserJoinedMsg struct {
	Room     string
	Username string
}

type UserLeftMsg struct {
	Room     string
	Username string
}

type RoomListMsg struct {
	Rooms []RoomEntry
}

type RoomEntry struct {
	Name      string
	UserCount int
}

type UserListMsg struct {
	Room  string
	Users []string
}

type ErrorMsg struct {
	Code    string
	Message string
}

type WsMessage struct {
	Msg tea.Msg
}
