package main

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
	"runtime"
	"time"
)

var teams = team.NewManager()

func main() {

	fmt.Println("===============================================")
	fmt.Println("  _____ _    _   ___  __")
	fmt.Println(" |  ___| |  | | | \\ \\/ /")
	fmt.Println(" | |_  | |  | | | |\\  / ")
	fmt.Println(" |  _| | |__| |_| |/  \\ ")
	fmt.Println(" |_|   |_____\\___//_/\\_\\")
	fmt.Println("")
	fmt.Println("===============================================")

	http.HandleFunc("/perf", perfHandler)

	network.Manager.HandleFunc("user:new", onUserJoin)
	network.Manager.HandleFunc("user:touch", onUserTouch)
	network.Manager.HandleFunc("user:touchEnd", onUserTouchEnd)
	network.Manager.HandleFunc("user:bloat", onUserBloat)
	network.Manager.HandleFunc("user:pinch", onUserPinch)
	network.Manager.HandleFunc("user:attack", onUserAttack)
	network.Manager.HandleFunc("user:disconnect", onUserDisconnect)

	network.Manager.HandleFunc("collector:merge", onCollectorMerge)

	go teams.Run()
	network.Init()
}

func onUserJoin(e events.Event) interface{} {
	// get incoming data in format of user.Id
	u := user.User{}
	if err := json.Unmarshal(e.Args, &u); err != nil {
		panic(err.Error())
	}

	if err := u.Save(); err != nil {
		panic(err)
	}

	// assign to team
	member := team.Member{User: u, Conn: e.Sender}
	teams.Queue <- member
	assignedTeamId := <-teams.LastId

	// forward to xna
	//simpleToXna("user:join", u.Id)
	msg := struct {
		Name   string `tcp:"name"`
		Id     int    `tcp:"id"`
		TeamId int    `tcp:"teamId"`
	}{"user:new", u.Id, assignedTeamId}
	network.TcpClients.Broadcast <- msg

	// reply to sencha with proper ID
	return packet.Out{
		Name:    "user:id",
		Message: user.Id{u.Id},
	}
}

func onUserDisconnect(e events.Event) interface{} {
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

	// forward to XNA
	msg := struct {
		Name string `tcp:"name"`
		Id   int    `tcp:"id"`
		X    int    `tcp:"x"`
		Y    int    `tcp:"y"`
	}{"user:touch", pos.Id, pos.X, pos.Y}

	network.TcpClients.Broadcast <- msg

	return nil
}

func onUserTouchEnd(e events.Event) interface{} {
	u := events.GetUserId(e)
	// forward to XNA
	simpleToXna("user:touchEnd", u.Id)
	return nil
}

func onUserBloat(e events.Event) interface{} {
	u := events.GetUserId(e)
	// forward to XNA
	simpleToXna("user:bloat", u.Id)
	return nil
}

func onUserPinch(e events.Event) interface{} {
	u := events.GetUserId(e)
	// forward to XNA
	simpleToXna("user:pinch", u.Id)
	return nil
}

func onUserAttack(e events.Event) interface{} {
	u := events.GetUserId(e)
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
		Teams        map[int][]team.Member
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

func onCollectorMerge(e events.Event) interface{} {

	toMerge := team.Merger{}
	if err := json.Unmarshal(e.Args, &toMerge); err != nil {
		panic(err.Error())
	}

	teams.Merge(toMerge)

	return nil
}

func simpleToXna(evt string, id int) interface{} {
	msg := struct {
		Name string `tcp:"name"`
		Id   int    `tcp:"id"`
	}{evt, id}

	network.TcpClients.Broadcast <- msg

	return nil
}
