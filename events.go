package main

import (
	"encoding/json"
)

type Event struct {
	Name string `json:"name"`
	Args json.RawMessage `json:"args"`
}

type UserId struct {
	Id int `json:"id"`
}

type Coords struct {
	Id int `json:"id"`
	X int `json:"x"`
	Y int `json:"y"`
}

type EventManager struct {
	incoming chan InPacket
	outgoing chan InPacket
	handlers map[string]eventHandlerFunc
}

func (em *EventManager) HandleFunc(pattern string, handler eventHandlerFunc) {
	em.handlers[pattern] = handler
}

func NewEventManager() EventManager {
	return EventManager{incoming: make(chan InPacket), outgoing: make(chan InPacket), handlers: make(map[string]eventHandlerFunc)}
}

type eventHandlerFunc func(packet InPacket)

// generates wrapper so that functions can manage a generic event
func makeEventHandler(fn func(user UserId)) eventHandlerFunc {
	return func(packet InPacket) {
		user := UserId{}
		if err := json.Unmarshal(packet.Args, &user); err != nil {
			panic(err.Error())
		}
		fn(user)
	}
}

// execute stored callbacks for each event received
func (em *EventManager) listen() {
	for packet := range em.incoming {
		if callback, exists := em.handlers[packet.Name]; exists {
			callback(packet)
		}
	}
}