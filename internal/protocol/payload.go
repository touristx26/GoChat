package protocol

// Client -> Server payloads

type SetUsernamePayload struct {
	Username string `json:"username"`
}

type JoinRoomPayload struct {
	Room string `json:"room"`
}

type LeaveRoomPayload struct {
	Room string `json:"room"`
}

type SendMessagePayload struct {
	Room    string `json:"room"`
	Content string `json:"content"`
}

type ListUsersPayload struct {
	Room string `json:"room"`
}

// Server -> Client payloads

type RoomInfo struct {
	Name      string `json:"name"`
	UserCount int    `json:"user_count"`
}

type WelcomePayload struct {
	UserID string     `json:"user_id"`
	Rooms  []RoomInfo `json:"rooms"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RoomJoinedPayload struct {
	Room  string   `json:"room"`
	Users []string `json:"users"`
}

type RoomLeftPayload struct {
	Room string `json:"room"`
}

type ChatMessagePayload struct {
	Room      string `json:"room"`
	From      string `json:"from"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

type UserJoinedPayload struct {
	Room     string `json:"room"`
	Username string `json:"username"`
}

type UserLeftPayload struct {
	Room     string `json:"room"`
	Username string `json:"username"`
}

type RoomListPayload struct {
	Rooms []RoomInfo `json:"rooms"`
}

type UserListPayload struct {
	Room  string   `json:"room"`
	Users []string `json:"users"`
}

type SystemMessagePayload struct {
	Room    string `json:"room"`
	Content string `json:"content"`
}
