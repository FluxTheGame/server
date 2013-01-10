package main

import (
	"bitbucket.org/jahfer/flux-middleman/events"
	"bitbucket.org/jahfer/flux-middleman/network"
	"bitbucket.org/jahfer/flux-middleman/packet"
	"bitbucket.org/jahfer/flux-middleman/user"
	"encoding/json"
	"fmt"
)

func main() {
	network.Manager.HandleFunc("user:join", events.Handler(onUserJoin))
	network.Manager.HandleFunc("user:bloat", events.Handler(onUserBloat))

	network.Init()
}

func onUserJoin(user user.User) {
	fmt.Println("-- Join", user)

	data, err := json.Marshal(user)
	if err != nil {
		panic(err.Error())
	}

	msg := packet.Out{
		Name:    "user:join",
		Message: string(data),
	}

	network.TcpClients.Broadcast <- msg
}

func onUserBloat(user user.User) {
	fmt.Println("[Bloat] User:", user)
}
