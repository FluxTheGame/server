package client

import (
	"fmt"
)

// Manager for incoming/outgoing traffic for a specified group of clients
type Hub struct {
	clients    map[Client]bool
	Broadcast  chan interface{}
	Register   chan Client
	Unregister chan Client
}

func NewHub() Hub {
	return Hub{
		clients:    make(map[Client]bool),
		Broadcast:  make(chan interface{}),
		Register:   make(chan Client),
		Unregister: make(chan Client),
	}
}

func (h Hub) NumClients() int {
	return len(h.clients)
}

// Manage traffic routing and (dis)connection of clients 
func (h *Hub) Run() {
	for {
		select {

		// add new client
		case c := <-h.Register:
			fmt.Printf("[NOTICE]\tclient connected\n")
			h.clients[c] = true

		// lost connection with client
		case c := <-h.Unregister:
			fmt.Printf("[NOTICE]\tclient disconnected\n")
			delete(h.clients, c)
			c.Close()

		// message being piped in to relay to clients
		case msg := <-h.Broadcast:
			for c := range h.clients {
				fmt.Printf("[SENDING]\t%+v\n", msg)
				// format according to protocol
				data := c.Format(msg)
				// send for transmit
				_, err := c.Write(data)
				if err != nil {
					fmt.Printf("[NOTICE]\tclient lost connection\n")
					delete(h.clients, c)
					c.Close()
				}
			}
			
		}
	}
}
