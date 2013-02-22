package events

import (
	"bitbucket.org/jahfer/flux-middleman/packet"
	"bitbucket.org/jahfer/flux-middleman/user"
	"encoding/json"
	"io"
)

type Event struct {
	Name string `json:"name"`
	Args []byte `json:"args"`
	Sender io.Writer
}

type eventHandlerFunc func(e Event) interface{}

type Manager struct {
	Incoming chan packet.In
	Outgoing chan packet.In
	handlers map[string]eventHandlerFunc
}

func NewManager() Manager {
	return Manager{
		Incoming: make(chan packet.In), 
		Outgoing: make(chan packet.In), 
		handlers: make(map[string]eventHandlerFunc),
	}
}

func (em *Manager) HandleFunc(pattern string, handler eventHandlerFunc) {
	em.handlers[pattern] = handler
}

// execute stored callbacks for each event received
func (em *Manager) Listener() {

	for pkt := range em.Incoming {

		// Dead packet; user has disconnected!
		if pkt.Raw == nil {
			// envoke disconnect callbacks
			if callback, exists := em.handlers["user:disconnect"]; exists {
				go callback(Event{ Name:"user:disconnect", Sender: pkt.Sender })
			}
			continue
		}

		// unmarshal incoming packet
		var e []Event
		if err := packet.Unmarshal(pkt.Raw, &e); err != nil {
			panic(err.Error())
		}
		evt := e[0]
		evt.Sender = pkt.Sender

		// envoke callback for event
		if callback, exists := em.handlers[evt.Name]; exists {
			response := callback(evt)
			// reply back to client
			if response != nil {
				data, err := json.Marshal(response)
				if err != nil {
					panic(err.Error())
				}
				evt.Sender.Write(data)
			}
		}
	}
}

// unmarshal an Event to strip just the user id information
func GetUserId(e Event) user.Id {
	u := user.Id{}
	if err := json.Unmarshal(e.Args, &u); err != nil {
		panic(err.Error())
	}

	return u
}
