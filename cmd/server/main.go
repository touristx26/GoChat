package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/xianfeng-wang/gochat/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "server listen address")
	flag.Parse()

	hub := server.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", server.HandleWebSocket(hub))

	fmt.Printf("GoChat server started on %s\n", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
