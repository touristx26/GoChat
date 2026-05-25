package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xianfeng-wang/gochat/internal/client"
)

func main() {
	addr := flag.String("addr", "ws://localhost:8080/ws", "server WebSocket address")
	username := flag.String("user", "", "your username")
	flag.Parse()

	if *username == "" {
		fmt.Print("Enter your username: ")
		fmt.Scanln(username)
		if *username == "" {
			fmt.Println("Username is required")
			os.Exit(1)
		}
	}

	ws := client.NewWSClient(*addr)
	if err := ws.Connect(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer ws.Close()

	m := client.NewModel(ws, *username)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
