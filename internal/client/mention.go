package client

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

var (
	popupBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("12")).
				Padding(0, 1)
	popupItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))
	popupSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("12")).
				Bold(true)
)

type MentionPopup struct {
	Active       bool
	TriggerPos   int
	Query        string
	Matches      []string
	SelectedIdx  int
	ScrollOffset int
	MaxVisible   int
}

func NewMentionPopup() MentionPopup {
	return MentionPopup{MaxVisible: 5}
}

func (p *MentionPopup) Open(triggerPos int, users []string, query string) {
	p.Active = true
	p.TriggerPos = triggerPos
	p.SelectedIdx = 0
	p.ScrollOffset = 0
	p.Filter(users, query)
}

func (p *MentionPopup) Close() {
	p.Active = false
	p.TriggerPos = 0
	p.Query = ""
	p.Matches = nil
	p.SelectedIdx = 0
	p.ScrollOffset = 0
}

func (p *MentionPopup) Filter(users []string, query string) {
	p.Query = query
	lower := strings.ToLower(query)
	p.Matches = p.Matches[:0]
	for _, u := range users {
		if strings.HasPrefix(strings.ToLower(u), lower) {
			p.Matches = append(p.Matches, u)
		}
	}
	if p.SelectedIdx >= len(p.Matches) {
		p.SelectedIdx = max(0, len(p.Matches)-1)
	}
	if p.ScrollOffset > p.SelectedIdx {
		p.ScrollOffset = p.SelectedIdx
	}
	if p.ScrollOffset+p.MaxVisible <= p.SelectedIdx {
		p.ScrollOffset = p.SelectedIdx - p.MaxVisible + 1
	}
}

func (p *MentionPopup) MoveUp() {
	if p.SelectedIdx > 0 {
		p.SelectedIdx--
	}
	if p.SelectedIdx < p.ScrollOffset {
		p.ScrollOffset = p.SelectedIdx
	}
}

func (p *MentionPopup) MoveDown() {
	if p.SelectedIdx < len(p.Matches)-1 {
		p.SelectedIdx++
	}
	if p.SelectedIdx >= p.ScrollOffset+p.MaxVisible {
		p.ScrollOffset = p.SelectedIdx - p.MaxVisible + 1
	}
}

func (p *MentionPopup) SelectedUser() string {
	if len(p.Matches) == 0 || p.SelectedIdx >= len(p.Matches) {
		return ""
	}
	return p.Matches[p.SelectedIdx]
}

func (p *MentionPopup) View() string {
	if !p.Active || len(p.Matches) == 0 {
		return ""
	}

	end := p.ScrollOffset + p.MaxVisible
	if end > len(p.Matches) {
		end = len(p.Matches)
	}
	visible := p.Matches[p.ScrollOffset:end]

	var sb strings.Builder
	for i, item := range visible {
		globalIdx := p.ScrollOffset + i
		line := "  " + item
		if globalIdx == p.SelectedIdx {
			line = "> " + item
			sb.WriteString(popupSelectedStyle.Render(line))
		} else {
			sb.WriteString(popupItemStyle.Render(line))
		}
		if i < len(visible)-1 {
			sb.WriteString("\n")
		}
	}

	return popupBorderStyle.Render(sb.String())
}

func shouldTriggerMention(value string, cursorPos int) (int, bool) {
	runes := []rune(value)
	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}

	for i := cursorPos - 1; i >= 0; i-- {
		ch := runes[i]
		if ch == '@' {
			if i == 0 || unicode.IsSpace(runes[i-1]) || unicode.IsPunct(runes[i-1]) {
				return i, true
			}
			return 0, false
		}
		if unicode.IsSpace(ch) {
			return 0, false
		}
	}
	return 0, false
}

func extractMentionQuery(value string, triggerPos int, cursorPos int) string {
	runes := []rune(value)
	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}
	if triggerPos+1 >= cursorPos {
		return ""
	}
	return string(runes[triggerPos+1 : cursorPos])
}
