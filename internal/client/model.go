package client

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xianfeng-wang/gochat/internal/protocol"
)

type ChatEntry struct {
	Timestamp string
	From      string
	Content   string
	IsSystem  bool
}

type Model struct {
	ws           *WSClient
	username     string
	userID       string
	currentRoom  string
	rooms        []RoomEntry
	users        []string
	messages     []ChatEntry
	viewport     viewport.Model
	textinput    textinput.Model
	mentionPopup MentionPopup
	width        int
	height       int
	connected    bool
	quitting     bool
	err          error
}

func NewModel(ws *WSClient, username string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message... (/help for commands)"
	ti.Focus()
	ti.CharLimit = 500

	vp := viewport.New(0, 0)

	return Model{
		ws:           ws,
		username:     username,
		textinput:    ti,
		viewport:     vp,
		mentionPopup: NewMentionPopup(),
		messages:     make([]ChatEntry, 0, 500),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.ws.ListenCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mentionPopup.Active {
			switch msg.Type {
			case tea.KeyUp:
				m.mentionPopup.MoveUp()
				return m, nil
			case tea.KeyDown:
				m.mentionPopup.MoveDown()
				return m, nil
			case tea.KeyTab:
				m.confirmMention()
				return m, nil
			case tea.KeyEsc:
				m.mentionPopup.Close()
				return m, nil
			}
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			return m.handleInput()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case ConnectedMsg:
		m.connected = true
		m.userID = msg.UserID
		m.addSystemMessage("Connected! Setting username...")
		m.ws.Send(protocol.TypeSetUsername, protocol.SetUsernamePayload{Username: m.username})
		m.ws.Send(protocol.TypeListRooms, struct{}{})
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case ChatMsg:
		m.messages = append(m.messages, ChatEntry{
			Timestamp: msg.Timestamp,
			From:      msg.From,
			Content:   msg.Content,
		})
		m.updateViewport()
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case RoomJoinedMsg:
		m.currentRoom = msg.Room
		m.users = msg.Users
		m.addSystemMessage(fmt.Sprintf("Joined #%s", msg.Room))
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case RoomLeftMsg:
		m.addSystemMessage(fmt.Sprintf("Left #%s", msg.Room))
		if m.currentRoom == msg.Room {
			m.currentRoom = ""
			m.users = nil
		}
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case UserJoinedMsg:
		if msg.Room == m.currentRoom {
			m.users = append(m.users, msg.Username)
			m.addSystemMessage(fmt.Sprintf("*** %s joined", msg.Username))
		}
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case UserLeftMsg:
		if msg.Room == m.currentRoom {
			m.removeUser(msg.Username)
			m.addSystemMessage(fmt.Sprintf("*** %s left", msg.Username))
		}
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case UsernameChangedMsg:
		if msg.Room == m.currentRoom {
			for i, u := range m.users {
				if u == msg.OldName {
					m.users[i] = msg.NewName
					break
				}
			}
			if msg.OldName == m.username {
				m.username = msg.NewName
			}
			m.addSystemMessage(fmt.Sprintf("*** %s is now known as %s", msg.OldName, msg.NewName))
		}
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case RoomListMsg:
		m.rooms = msg.Rooms
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case UserListMsg:
		if msg.Room == m.currentRoom {
			m.users = msg.Users
		}
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case ErrorMsg:
		m.addSystemMessage(fmt.Sprintf("[Error] %s", msg.Message))
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case SystemMsg:
		m.addSystemMessage(msg.Content)
		cmds = append(cmds, m.ws.ListenCmd())
		return m, tea.Batch(cmds...)

	case DisconnectedMsg:
		m.connected = false
		m.addSystemMessage("Disconnected from server")
		return m, nil
	}

	if m.quitting {
		return m, tea.Quit
	}

	var tiCmd tea.Cmd
	m.textinput, tiCmd = m.textinput.Update(msg)
	cmds = append(cmds, tiCmd)

	m.detectMention()

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) handleInput() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textinput.Value())
	if input == "" {
		return m, nil
	}
	m.textinput.Reset()
	m.mentionPopup.Close()

	if cmd, ok := ParseCommand(input); ok {
		m.executeCommand(cmd)
		if m.quitting {
			return m, tea.Quit
		}
		return m, m.ws.ListenCmd()
	}

	if m.currentRoom == "" {
		m.addSystemMessage("Join a room first: /join <room>")
		return m, nil
	}

	m.ws.Send(protocol.TypeSendMessage, protocol.SendMessagePayload{
		Room:    m.currentRoom,
		Content: input,
	})
	return m, nil
}

func (m *Model) addSystemMessage(content string) {
	m.messages = append(m.messages, ChatEntry{
		Content:  content,
		IsSystem: true,
	})
	m.updateViewport()
}

func (m *Model) removeUser(username string) {
	for i, u := range m.users {
		if u == username {
			m.users = append(m.users[:i], m.users[i+1:]...)
			return
		}
	}
}

func (m *Model) detectMention() {
	value := m.textinput.Value()
	cursorPos := m.textinput.Position()
	wasActive := m.mentionPopup.Active

	triggerPos, ok := shouldTriggerMention(value, cursorPos)
	if ok {
		query := extractMentionQuery(value, triggerPos, cursorPos)
		if !m.mentionPopup.Active || m.mentionPopup.TriggerPos != triggerPos {
			m.mentionPopup.Open(triggerPos, m.filteredUsers(), query)
		} else {
			m.mentionPopup.Filter(m.filteredUsers(), query)
		}
		if len(m.mentionPopup.Matches) == 0 {
			m.mentionPopup.Close()
		}
	} else if m.mentionPopup.Active {
		m.mentionPopup.Close()
	}

	if wasActive != m.mentionPopup.Active {
		m.updateLayout()
	}
}

func (m *Model) confirmMention() {
	selected := m.mentionPopup.SelectedUser()
	if selected == "" {
		m.mentionPopup.Close()
		return
	}

	value := m.textinput.Value()
	runes := []rune(value)
	cursorPos := m.textinput.Position()
	triggerPos := m.mentionPopup.TriggerPos

	replacement := []rune("@" + selected + " ")
	newRunes := make([]rune, 0, len(runes)+len(replacement))
	newRunes = append(newRunes, runes[:triggerPos]...)
	newRunes = append(newRunes, replacement...)
	newRunes = append(newRunes, runes[cursorPos:]...)

	m.textinput.SetValue(string(newRunes))
	m.textinput.SetCursor(triggerPos + len(replacement))
	m.mentionPopup.Close()
}

func (m *Model) filteredUsers() []string {
	result := make([]string, 0, len(m.users))
	for _, u := range m.users {
		if u != m.username {
			result = append(result, u)
		}
	}
	return result
}

func (m *Model) updateLayout() {
	sidebarWidth := 24
	chatWidth := m.width - sidebarWidth - 3
	chatHeight := m.height - 5

	if chatWidth < 20 {
		chatWidth = m.width - 2
	}
	if chatHeight < 3 {
		chatHeight = 3
	}

	if m.mentionPopup.Active && len(m.mentionPopup.Matches) > 0 {
		visible := min(m.mentionPopup.MaxVisible, len(m.mentionPopup.Matches))
		popupHeight := visible + 2
		chatHeight -= popupHeight
		if chatHeight < 3 {
			chatHeight = 3
		}
	}

	m.viewport.Width = chatWidth
	m.viewport.Height = chatHeight
	m.textinput.Width = m.width - 4
	m.updateViewport()
}

func (m *Model) updateViewport() {
	var sb strings.Builder
	mentionPattern := "@" + m.username
	for _, msg := range m.messages {
		if msg.IsSystem {
			sb.WriteString(systemStyle.Render(msg.Content))
		} else {
			ts := dimStyle.Render(fmt.Sprintf("[%s]", msg.Timestamp))
			name := nameStyle.Render(msg.From + ":")
			line := fmt.Sprintf("%s %s %s", ts, name, msg.Content)
			if strings.Contains(msg.Content, mentionPattern) {
				line = mentionHighlightStyle.Render(
					fmt.Sprintf("[%s] %s: %s", msg.Timestamp, msg.From, msg.Content),
				)
			}
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

// --- View ---

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	borderStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	nameStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	systemStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Italic(true)
	activeRoom   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	inactiveRoom = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	statusOnline = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	statusOffline = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	mentionHighlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("3")).Foreground(lipgloss.Color("0"))
)

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	if m.width == 0 {
		return "Loading...\n"
	}

	header := m.renderHeader()
	sidebar := m.renderSidebar()
	chat := m.renderChat()
	input := m.renderInput()
	popupView := m.mentionPopup.View()

	sidebarWidth := 24
	chatWidth := m.width - sidebarWidth - 3

	if chatWidth < 20 {
		body := lipgloss.JoinVertical(lipgloss.Left, chat)
		if popupView != "" {
			return lipgloss.JoinVertical(lipgloss.Left, header, body, popupView, input)
		}
		return lipgloss.JoinVertical(lipgloss.Left, header, body, input)
	}

	chatPanel := lipgloss.NewStyle().Width(chatWidth).Render(chat)
	sidePanel := lipgloss.NewStyle().Width(sidebarWidth).Render(sidebar)
	body := lipgloss.JoinHorizontal(lipgloss.Top, chatPanel, " ", sidePanel)

	if popupView != "" {
		return lipgloss.JoinVertical(lipgloss.Left, header, body, popupView, input)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, input)
}

func (m Model) renderHeader() string {
	status := statusOnline.Render("[Connected]")
	if !m.connected {
		status = statusOffline.Render("[Disconnected]")
	}

	room := "none"
	if m.currentRoom != "" {
		room = "#" + m.currentRoom
	}

	title := titleStyle.Render("GoChat")
	info := fmt.Sprintf(" - Room: %s  User: %s  %s", room, m.username, status)
	return title + dimStyle.Render(info)
}

func (m Model) renderSidebar() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Rooms") + "\n")
	if len(m.rooms) == 0 {
		sb.WriteString(dimStyle.Render("  (none)") + "\n")
	}
	for _, r := range m.rooms {
		prefix := "  "
		style := inactiveRoom
		if r.Name == m.currentRoom {
			prefix = "> "
			style = activeRoom
		}
		sb.WriteString(style.Render(fmt.Sprintf("%s#%s (%d)", prefix, r.Name, r.UserCount)) + "\n")
	}

	sb.WriteString("\n" + titleStyle.Render("Users") + "\n")
	if len(m.users) == 0 {
		sb.WriteString(dimStyle.Render("  (none)") + "\n")
	}
	for _, u := range m.users {
		suffix := ""
		if u == m.username {
			suffix = " (you)"
		}
		sb.WriteString(fmt.Sprintf("  %s%s\n", u, dimStyle.Render(suffix)))
	}

	return sb.String()
}

func (m Model) renderChat() string {
	return m.viewport.View()
}

func (m Model) renderInput() string {
	return m.textinput.View()
}
