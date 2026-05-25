package server

import "sync"

type Room struct {
	Name    string
	members map[*Client]bool
	mu      sync.RWMutex
}

func NewRoom(name string) *Room {
	return &Room{
		Name:    name,
		members: make(map[*Client]bool),
	}
}

func (r *Room) Join(c *Client) {
	r.mu.Lock()
	r.members[c] = true
	r.mu.Unlock()
}

func (r *Room) Leave(c *Client) {
	r.mu.Lock()
	delete(r.members, c)
	r.mu.Unlock()
}

func (r *Room) Members() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	clients := make([]*Client, 0, len(r.members))
	for c := range r.members {
		clients = append(clients, c)
	}
	return clients
}

func (r *Room) UserCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.members)
}

func (r *Room) Broadcast(data []byte, exclude *Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for c := range r.members {
		if c != exclude {
			c.Send(data)
		}
	}
}

func (r *Room) Usernames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.members))
	for c := range r.members {
		names = append(names, c.Username)
	}
	return names
}
