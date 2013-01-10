package events

import (
	"encoding/json"
	"bitbucket.org/jahfer/flux-middleman/packet"
	"bitbucket.org/jahfer/flux-middleman/user"
)

type Event struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type Manager struct {
	Incoming chan packet.In
	Outgoing chan packet.In
	handlers map[string]eventHandlerFunc
}

func (em *Manager) HandleFunc(pattern string, handler eventHandlerFunc) {
	em.handlers[pattern] = handler
}

func NewManager() Manager {
	return Manager{
		Incoming: make(chan packet.In), 
		Outgoing: make(chan packet.In), 
		handlers: make(map[string]eventHandlerFunc),
	}
}

// execute stored callbacks for each event received
func (em *Manager) Listener() {
	for p := range em.Incoming {

		/*
		 unmarshal all incoming packets here
		 then dispatch in event format
		*/

		var e []Event

		if err := packet.Unmarshal(p.Raw, &e); err != nil {
			panic(err.Error())
		}
		evt := e[0]

		if callback, exists := em.handlers[evt.Name]; exists {
			callback(evt)
		}
	}
}

type eventHandlerFunc func(e Event)

// generates wrapper so that functions can manage a generic event
func Handler(fn func(user user.User)) eventHandlerFunc {
	return func(e Event) {

		user := user.User{}

		if err := json.Unmarshal(e.Args, &user); err != nil {
			panic(err.Error())
		}
		fn(user)
	}
}
