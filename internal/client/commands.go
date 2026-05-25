package client

import (
	"strings"

	"github.com/xianfeng-wang/gochat/internal/protocol"
)

type Command struct {
	Name string
	Args string
}

func ParseCommand(input string) (Command, bool) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return Command{}, false
	}
	parts := strings.SplitN(input[1:], " ", 2)
	cmd := Command{Name: strings.ToLower(parts[0])}
	if len(parts) > 1 {
		cmd.Args = strings.TrimSpace(parts[1])
	}
	return cmd, true
}

func (m *Model) executeCommand(cmd Command) {
	switch cmd.Name {
	case "join":
		if cmd.Args == "" {
			m.addSystemMessage("Usage: /join <room>")
			return
		}
		m.ws.Send(protocol.TypeJoinRoom, protocol.JoinRoomPayload{Room: cmd.Args})

	case "leave":
		if m.currentRoom == "" {
			m.addSystemMessage("Not in any room")
			return
		}
		room := m.currentRoom
		if cmd.Args != "" {
			room = cmd.Args
		}
		m.ws.Send(protocol.TypeLeaveRoom, protocol.LeaveRoomPayload{Room: room})

	case "rooms":
		m.ws.Send(protocol.TypeListRooms, struct{}{})

	case "users":
		if m.currentRoom == "" {
			m.addSystemMessage("Not in any room")
			return
		}
		m.ws.Send(protocol.TypeListUsers, protocol.ListUsersPayload{Room: m.currentRoom})

	case "nick":
		if cmd.Args == "" {
			m.addSystemMessage("Usage: /nick <username>")
			return
		}
		m.username = cmd.Args
		m.ws.Send(protocol.TypeSetUsername, protocol.SetUsernamePayload{Username: cmd.Args})
		m.addSystemMessage("Username set to: " + cmd.Args)

	case "help":
		m.addSystemMessage("Commands: /join <room>, /leave, /rooms, /users, /nick <name>, /quit")

	case "quit":
		m.quitting = true

	default:
		m.addSystemMessage("Unknown command: /" + cmd.Name + " (type /help)")
	}
}
