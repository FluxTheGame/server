package helper

import (
	"bitbucket.org/jahfer/flux-middleman/network"
	"bitbucket.org/jahfer/flux-middleman/db"
	"fmt"
)

func SendBadge(badge string, userId int) {
	msg := struct {
		Name string `tcp:"name"`
		Badge string `tcp:"type"`
		Id   int    `tcp:"id"`
	}{"user:getBadge", badge, userId}

	badgeKey := fmt.Sprintf("uid:%v:badges", userId)
	res := db.Redis.SAdd(badgeKey, badge)

	if res.Val() != 0 {
		network.TcpClients.Broadcast <- msg
	}
}

func SendPoints(amount, userId int) {
	msg := struct {
		Name 	string `tcp:"name"`
		Value 	int `tcp:"value"`
		Id   	int    `tcp:"id"`
	}{"user:getPoints", amount, userId}

	network.TcpClients.Broadcast <- msg
}

func ToXna(evt string, id int) {
	msg := struct {
		Name string `tcp:"name"`
		Id   int    `tcp:"id"`
	}{evt, id}

	network.TcpClients.Broadcast <- msg
}