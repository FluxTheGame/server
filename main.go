package main

import (
	"bitbucket.org/jahfer/flux-middleman/db"
	r "github.com/vmihailenco/redis"
	"bitbucket.org/jahfer/flux-middleman/events"
	"bitbucket.org/jahfer/flux-middleman/helper"
	"bitbucket.org/jahfer/flux-middleman/network"
	"bitbucket.org/jahfer/flux-middleman/packet"
	"bitbucket.org/jahfer/flux-middleman/tcp"
	"bitbucket.org/jahfer/flux-middleman/team"
	"bitbucket.org/jahfer/flux-middleman/user"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"time"
	"strconv"
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
	network.Manager.HandleFunc("user:bloatEnd", onUserBloatEnd)
	network.Manager.HandleFunc("user:pinch", onUserPinch)
	network.Manager.HandleFunc("user:pinchEnd", onUserPinchEnd)
	network.Manager.HandleFunc("user:attack", onUserAttack)
	network.Manager.HandleFunc("user:disconnect", onUserDisconnect)
	network.Manager.HandleFunc("user:heartbeat", onUserHeartbeat)

	network.Manager.HandleFunc("collector:merge", onCollectorMerge)
	network.Manager.HandleFunc("collector:burst", onCollectorBurst)

	// get every client that has been dead for at least 100 seconds...stupid Go precision
	go func() {
		ticker := time.NewTicker(5 * time.Second)

		for {
			select {
			case <-ticker.C:
				expired := db.Redis.ZRangeByScore("global:clients", "-inf", strconv.FormatInt(time.Now().Unix() - 10, 10), 0, -1).Val()
				if len(expired) > 0 {
					fmt.Printf("EXPIRED USERS: %v\n", expired)

					for _, idStr := range expired {
						db.Redis.ZRem("global:clients", idStr)
						id, _ := strconv.Atoi(idStr)
						teamId, _ := strconv.Atoi(db.Redis.Get("uid:"+idStr+":team").Val())
						userIndex := teams.GetUserIndex(teamId, id)
						if (userIndex == -1) {
							fmt.Printf("[ERROR]\tUser's index out of bounds\n")
							continue
						}
						teams.RemoveMember(teamId, id, userIndex)
					}
				}
			}
		}
	}()

	go teams.Run()
	network.Init()
}

func onUserJoin(e events.Event) interface{} {

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ERROR]\t%v\n", r)
		}
	}()

	// get incoming data in format of user.Id
	u := user.User{}
	if err := json.Unmarshal(e.Args, &u); err != nil {
		fmt.Printf("[ERROR]\tCould not unmarshal new user. " + err.Error() + "\n")
	}

	if err, err2 := u.Save(); err != nil || err2 != nil {
		fmt.Printf("[ERROR]\tCould not save user. " + err.Error() + " " + err2.Error() + "\n")
	}

	// assign to team
	member := team.Member{User: u, Conn: e.Sender}
	teams.Queue <- member
	// get team id - blocking
	assignedTeamId := <-teams.LastId
	member.User.TeamId = assignedTeamId

	// forward to xna
	msg := struct {
		Name     string `tcp:"name"`
		Id       int    `tcp:"id"`
		Username string `tcp:"username"`
		TeamId   int    `tcp:"teamId"`
	}{"user:new", u.Id, u.Name, assignedTeamId}
	network.TcpClients.Broadcast <- msg

	helper.SendBadge("join", u.Id)

	// reply to sencha with proper ID
	return packet.Out{
		Name:    "user:info",
		Message: member.User,
	}
}

func onUserHeartbeat(e events.Event) interface{} {
	u := events.GetUserId(e)

	score := float64(time.Now().Unix())

	heartbeat := r.Z{
		score,
		strconv.Itoa(u.Id),
	}

	db.Redis.ZAdd("global:clients", heartbeat)
	
	return nil
}

func onUserDisconnect(e events.Event) interface{} {
	_, id, _ := teams.GetIndex(e.Sender)

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

func onUserBloatEnd(e events.Event) interface{} {
	u := events.GetUserId(e)
	// forward to XNA
	simpleToXna("user:bloatEnd", u.Id)
	return nil
}

func onUserPinch(e events.Event) interface{} {
	u := events.GetUserId(e)
	// forward to XNA
	simpleToXna("user:pinch", u.Id)
	return nil
}

func onUserPinchEnd(e events.Event) interface{} {
	u := events.GetUserId(e)
	// forward to XNA
	simpleToXna("user:pinchEnd", u.Id)
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
	// e.g. /name=collector:merge/team_1=0/team_2=1$

	fmt.Printf("%+v\n", e)

	toMerge := team.Merger{}
	if err := tcp.Unmarshal(e.Args, &toMerge); err != nil {
		panic(err.Error())
	}

	teams.Merge(toMerge)

	return nil
}

func onCollectorBurst(e events.Event) interface{} {
	// e.g. /name=collector:burst/id=0/points=155$

	type collector struct {
		Name   string `tcp:"name"`
		Id     int    `tcp:"id"`
		Points int    `tcp:"points"`
	}
	c := collector{}
	tcp.Unmarshal(e.Args, &c)

	if team, ok := teams.Roster[c.Id]; ok {
		pts := c.Points / len(team)

		for _, member := range team {
			userKey := fmt.Sprintf("uid:%v:points", member.User.Id)
			helper.SendBadge("firstComplete", member.User.Id)
			totalPts := db.Redis.IncrBy(userKey, int64(pts))
			helper.SendPoints(pts, member.User.Id)

			toApp := packet.Out{
				Name:    "user:getPoints",
				Message: totalPts.Val(),
			}

			encoded, err := json.Marshal(toApp)
			if err != nil {
				panic(err)
			}

			member.Conn.Write(encoded)
		}

		teams.ReturnToQueue(c.Id)
	}

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
