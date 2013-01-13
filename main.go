package main

/* ----| TODO ----------------- */
// 1. Logic for team sorting
// 2. Disconnection event to XNA
// 3. Dispatch events for incoming TCP packet

import (
	"bitbucket.org/jahfer/flux-middleman/events"
	"bitbucket.org/jahfer/flux-middleman/network"
	"bitbucket.org/jahfer/flux-middleman/packet"
	"bitbucket.org/jahfer/flux-middleman/team"
	"bitbucket.org/jahfer/flux-middleman/user"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

const DIR_TMPL = "tmpl/"

var templates = template.Must(template.ParseFiles(DIR_TMPL + "perf.html"))

var teams = team.NewManager()

// Spit out statistics for entire program
func perfHandler(w http.ResponseWriter, r *http.Request) {

	data := struct {
		WsNumConn  int
		TcpNumConn int
		NumTeams   int
		NumInQueue int
		NumActive  int
	}{
		WsNumConn:  network.WsClients.NumClients(),
		TcpNumConn: network.TcpClients.NumClients(),
		NumTeams:   len(teams.Roster),
		NumInQueue: len(teams.Queue),
		NumActive:  teams.NumUsers(),
	}

	err := templates.ExecuteTemplate(w, "perf.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {

	http.HandleFunc("/perf", perfHandler)

	network.Manager.HandleFunc("user:join", onUserJoin)
	network.Manager.HandleFunc("user:touch", onUserTouch)
	network.Manager.HandleFunc("user:touchEnd", onUserTouchEnd)
	network.Manager.HandleFunc("user:bloat", onUserBloat)
	network.Manager.HandleFunc("user:pinch", onUserPinch)
	network.Manager.HandleFunc("user:attack", onUserAttack)
	network.Manager.HandleFunc("disconnect", onDisconnect)

	go teams.Run()
	network.Init()
}

func onUserJoin(e events.Event) {
	// get incoming data in format of user.Id
	u := user.Id{}
	if err := json.Unmarshal(e.Args, &u); err != nil {
		panic(err.Error())
	}

	fmt.Println("-- Join", u)
	id := user.LastId()

	simpleToXna("user:join", id)

	// reply to sencha with new ID
	userIdSet := packet.Out{
		Name:    "server:createId",
		Message: user.Id{id},
	}
	data, err := json.Marshal(userIdSet)
	if err != nil {
		panic(err.Error())
	}
	e.Sender.Write(data)

	member := team.Member{Id: id, Conn: e.Sender}
	teams.Queue <- member
}

func onDisconnect(e events.Event) {
	teams.Unregister <- e.Sender
}

func onUserTouch(e events.Event) {
	// get incoming data in format of user.Coords
	pos := user.Coords{}
	if err := json.Unmarshal(e.Args, &pos); err != nil {
		panic(err.Error())
	}

	fmt.Printf("(user:touch)\t#%d: (%d, %d)\n", pos.Id, pos.X, pos.Y)

	// forward to XNA
	msg := struct {
		Name     string
		Id, X, Y int
	}{"user:touch", pos.Id, pos.X, pos.Y}

	network.TcpClients.Broadcast <- msg
}

func onUserBloat(e events.Event) {
	u := getUserId(e)
	fmt.Printf("(user:bloat)\t#%d\n", u.Id)

	// forward to XNA
	simpleToXna("user:bloat", u.Id)
}

func onUserPinch(e events.Event) {
	u := getUserId(e)
	fmt.Printf("(user:pinch)\t#%d\n", u.Id)

	// forward to XNA
	simpleToXna("user:pinch", u.Id)
}

func onUserAttack(e events.Event) {
	u := getUserId(e)
	fmt.Printf("(user:attack)\t#%d\n", u.Id)

	// forward to XNA
	simpleToXna("user:attack", u.Id)
}

func onUserTouchEnd(e events.Event) {
	u := getUserId(e)
	fmt.Printf("(user:touchEnd)\t#%d\n", u.Id)

	// forward to XNA
	simpleToXna("user:touchEnd", u.Id)
}

func simpleToXna(evt string, id int) {
	msg := struct {
		Name string
		Id   int
	}{evt, id}

	network.TcpClients.Broadcast <- msg
}

func getUserId(e events.Event) user.Id {
	u := user.Id{}
	if err := json.Unmarshal(e.Args, &u); err != nil {
		panic(err.Error())
	}

	return u
}
