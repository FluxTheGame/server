package team

import (
	"bitbucket.org/jahfer/flux-middleman/user"
	"bitbucket.org/jahfer/flux-middleman/client"
	"math"
	"fmt"
)

var LastId int = 0

type Team struct {
	Id int
	Members []user.User
}

// Manages all of the teams
type TeamHub struct {
	Hubs []client.Hub
	Broadcast chan interface{}
	Register chan client.Client
	Unregister chan client.Client
}

// Generate new TeamHub
func NewHub() TeamHub {
	t := TeamHub{ 
		Hubs: make([]client.Hub, 256),
		Broadcast:  make(chan interface{}),
		Register:   make(chan client.Client),
		Unregister: make(chan client.Client),
	}
	t.Hubs[0] = client.NewHub()

	return t
}

// Shows number of open connections, not logged-in users
func (t TeamHub) NumUsers() int {
	var count int

	for _, h := range t.Hubs {
		count += h.NumClients()
	}

	return count
}

func (t TeamHub) MaxTeams() int {
	userCount := float64(t.NumUsers())
	max := math.Ceil( math.Sqrt(userCount) )
	return int(max)
}

func (t *TeamHub) Sort() {

}

// Boot cycle for team manager
func (t *TeamHub) Run() {
	for _, h := range t.Hubs {
		// make sure all the team collections are running (aka 1 at init)
		go h.Run()
	}

	for {
		select {

		// add new client
		case c := <-t.Register:
			hub := t.getNextHub()
			/* todo: logic for hub sorting */
			hub.Register <- c
			LastId++

		// lost connection with client
		case c := <-t.Unregister:
			/* todo: dynamically find hub associated with connection…a map perhaps? */
			t.Hubs[0].Unregister <- c

		// send message to all phones
		case msg := <-t.Broadcast:
			for _, h := range t.Hubs {
				h.Broadcast <- msg
			}

		}
	}
}

// Find a hub to place the currently-queued player
func (t *TeamHub) getNextHub() client.Hub {

	var h client.Hub

	if len(t.Hubs) < t.MaxTeams() {
		fmt.Println("Creating a new hub...")
		h = t.Hubs[len(t.Hubs)]
		h = client.NewHub()
	} else {
		fmt.Println("Adding users to existing hub...")
		h = t.Hubs[0]
	}

	fmt.Println(len(t.Hubs), "hubs active")
	return h
}