package main

import (
	"bitbucket.org/jahfer/flux-middleman/network"
	"bitbucket.org/jahfer/flux-middleman/events"
	"bitbucket.org/jahfer/flux-middleman/packet"
	"bitbucket.org/jahfer/flux-middleman/team"
	"bitbucket.org/jahfer/flux-middleman/user"
	"encoding/json"
	"html/template"
	"net/http"
	"runtime"
	"fmt"
)

var teams = team.NewManager()

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

func onUserJoin(e events.Event) interface{} {
	// get incoming data in format of user.Id
	u := user.User{}
	if err := json.Unmarshal(e.Args, &u); err != nil {
		panic(err.Error())
	}
	u.Id = user.LastId()

	fmt.Printf("-- Join #%d\n", u.Id)

	// forward to xna
	simpleToXna("user:join", u.Id)

	// assign to team
	member := team.Member{User: u, Conn: e.Sender}
	teams.Queue <- member

	// reply to sencha with proper ID
	return packet.Out{
		Name:    "server:createId",
		Message: user.Id{u.Id},
	}
}

func onDisconnect(e events.Event) interface{} {
	_, id := teams.GetIndex(e.Sender)
	
	teams.Unregister <- e.Sender

	simpleToXna("user:disconnect", id)
	return nil
}

func onUserTouch(e events.Event) interface{} {
	// get incoming data in format of user.Coords
	pos := user.Coords{}
	if err := json.Unmarshal(e.Args, &pos); err != nil {
		panic(err.Error())
	}

	fmt.Printf("(user:touch)\t#%d: (%d, %d)\n", pos.Id, pos.X, pos.Y)

	// forward to XNA
	msg := struct {
		Name    string 	`tcp:"name"`
		Id 		int 	`tcp:"id"`
		X 		int 	`tcp:"x"`
		Y 		int 	`tcp:"y"`
	}{"user:touch", pos.Id, pos.X, pos.Y}

	network.TcpClients.Broadcast <- msg

	return nil
}

func onUserTouchEnd(e events.Event) interface{} {
	u := events.GetUserId(e)
	fmt.Printf("(user:touchEnd)\t#%d\n", u.Id)

	// forward to XNA
	simpleToXna("user:touchEnd", u.Id)
	return nil
}

func onUserBloat(e events.Event) interface{} {
	u := events.GetUserId(e)
	fmt.Printf("(user:bloat)\t#%d\n", u.Id)

	// forward to XNA
	simpleToXna("user:bloat", u.Id)
	return nil
}

func onUserPinch(e events.Event) interface{} {
	u := events.GetUserId(e)
	fmt.Printf("(user:pinch)\t#%d\n", u.Id)

	// forward to XNA
	simpleToXna("user:pinch", u.Id)
	return nil
}

func onUserAttack(e events.Event) interface{} {
	u := events.GetUserId(e)
	fmt.Printf("(user:attack)\t#%d\n", u.Id)

	// forward to XNA
	simpleToXna("user:attack", u.Id)
	return nil
}

// Spit out performance statistics for entire program
func perfHandler(w http.ResponseWriter, r *http.Request) {

	data := struct {
		NumGoroutine int
		WsNumConn    int
		TcpNumConn   int
		NumTeams     int
		NumInQueue   int
		NumActive    int
		Teams        [][]team.Member
	}{
		NumGoroutine: runtime.NumGoroutine(),
		WsNumConn:    network.WsClients.NumClients(),
		TcpNumConn:   network.TcpClients.NumClients(),
		NumTeams:     len(teams.Roster),
		NumInQueue:   len(teams.Queue),
		NumActive:    teams.NumUsers(),
		Teams:        teams.Roster,
	}

	t, _ := template.ParseFiles("tmpl/perf.html")
    err := t.Execute(w, data)
    if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func simpleToXna(evt string, id int) interface{} {
	msg := struct {
		Name string `tcp:"name"`
		Id   int `tcp:"id"`
	}{evt, id}

	network.TcpClients.Broadcast <- msg

	return nil
}