package main

import (
	"fmt"
)

// Manager for incoming/outgoing traffic for a specified group of clients
type hub struct {
	clients map[client]bool
	broadcast chan interface{}
	register chan client
	unregister chan client
}

func NewHub() hub {
	return hub{
		broadcast: 	make(chan interface{}),
		register:	make(chan client),
		unregister:	make(chan client),
		clients:	make(map[client]bool),
	}
}

// Manage traffic routing and (dis)connection of clients 
func (h *hub) run() {
	for {
		select {
		// add new client
		case c := <-h.register:
			h.clients[c] = true
		// lost connection with client
		case c := <-h.unregister:
			fmt.Println("-- client disconnected")
			delete(h.clients, c)
			c.Close()
		// message being piped in to relay to clients
		case msg := <-h.broadcast:
			for c := range h.clients {
				// format according to protocol
				data := c.Format(msg)
				// send for transmit
				_, err := c.Write(data)
				if err != nil {
					fmt.Println("-- client lost connection")
					delete(h.clients, c)
					c.Close()
				}
			}
		}
	}
}