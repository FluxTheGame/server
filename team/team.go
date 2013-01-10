package team

import (
	"bitbucket.org/jahfer/flux-middleman/user"
	"bitbucket.org/jahfer/flux-middleman/client"
	"math"
)

var LastId int = 0

type Team struct {
	Id int
	Members []user.User
}

type TeamHub struct {
	Hubs []client.Hub
	Broadcast chan interface{}
	Register chan client.Client
	Unregister chan client.Client
}

func NewHub() TeamHub {
	t := TeamHub{ 
		Hubs: make([]client.Hub, 1),
		Broadcast:  make(chan interface{}),
		Register:   make(chan client.Client),
		Unregister: make(chan client.Client),
	}
	t.Hubs[0] = client.NewHub()

	return t
}

func (t TeamHub) NumUsers() int {
	var count int

	for _, h := range t.Hubs {
		count += h.NumClients()
	}

	return count
}

func (t TeamHub) AvgNumUsers() int {
	return t.NumUsers() / len(t.Hubs)
}

func (t TeamHub) MaxTeams() int {
	userCount := float64(t.NumUsers())
	max := math.Ceil( math.Sqrt(userCount) )
	return int(max)
}

func (t *TeamHub) Run() {
	for _, h := range t.Hubs {
		go h.Run()
	}

	for {
		select {
		// add new client
		case c := <-t.Register:
			hub := t.getNextHub()
			hub.Register <- c
			LastId++
		// lost connection with client
		case c := <-t.Unregister:
			t.Hubs[0].Unregister <- c
		case msg := <-t.Broadcast:
			for _, h := range t.Hubs {
				h.Broadcast <- msg
			}
		}
	}
}

// Find a hub to place the 
func (t *TeamHub) getNextHub() client.Hub {
	return t.Hubs[0]
}