package main

import (
	"bitbucket.org/jahfer/flux-middleman/db"
	"bitbucket.org/jahfer/flux-middleman/events"
	"bitbucket.org/jahfer/flux-middleman/helper"
	"bitbucket.org/jahfer/flux-middleman/network"
	"bitbucket.org/jahfer/flux-middleman/client"
	"bitbucket.org/jahfer/flux-middleman/packet"
	"bitbucket.org/jahfer/flux-middleman/tcp"
	"bitbucket.org/jahfer/flux-middleman/team"
	"bitbucket.org/jahfer/flux-middleman/user"
	"encoding/json"
	"fmt"
	r "github.com/vmihailenco/redis"
	"html/template"
	"net/http"
	"runtime"
	"strconv"
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

	http.HandleFunc("/perf", performanceHandler)

	network.Manager.HandleFunc("user:new", onUserJoin)
	network.Manager.HandleFunc("user:touch", onUserTouch)
	network.Manager.HandleFunc("user:disconnect", onUserDisconnect)
	network.Manager.HandleFunc("user:heartbeat", onUserHeartbeat)

	network.Manager.HandleFunc("user:touchEnd", forwardEvent("user:touchEnd"))
	network.Manager.HandleFunc("user:bloat", forwardEvent("user:bloat"))
	network.Manager.HandleFunc("user:bloatEnd", forwardEvent("user:bloatEnd"))
	network.Manager.HandleFunc("user:pinch", forwardEvent("user:pinch"))
	network.Manager.HandleFunc("user:pinchEnd", forwardEvent("user:pinchEnd"))
	network.Manager.HandleFunc("user:attack", onUserAttack)

	network.Manager.HandleFunc("collector:merge", onCollectorMerge)
	network.Manager.HandleFunc("collector:burst", onCollectorBurst)
	network.Manager.HandleFunc("collector:heartbeat", onCollectorHeartbeat)

	go teams.Run()
	go cleanup()

	network.Init()
}

func cleanup() {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			teams.CheckExpired()
			teams.Analytics()
		}
	}
}

func forwardEvent(evtName string) func(e events.Event) interface{} {
	return func(e events.Event) interface{} {
		u := events.GetUserId(e)
		// forward to XNA
		helper.ToXna(evtName, u.Id)
		return nil
	}
}

func onUserAttack(e events.Event) interface{} {
	u := events.GetUserId(e)

	userShotKey := fmt.Sprintf("uid:%v:shotsFired", u.Id)
	db.Redis.Incr(userShotKey)

	// forward to XNA
	helper.ToXna("user:attack", u.Id)
	return nil
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

	u.Name = client.Sanitizer.ReplaceAllString(u.Name, "")

	if err, err2 := u.Save(); err != nil || err2 != nil {
		fmt.Printf("[ERROR]\tCould not save user. " + err.Error() + " " + err2.Error() + "\n")
	}

	// assign to team
	member := team.Member{User: u, Conn: e.Sender}
	teams.Queue <- member
	// get team id - blocking
	assignedTeamId := <-teams.LastId

	// forward to xna
	msg := struct {
		Name     string `tcp:"name"`
		Id       int    `tcp:"id"`
		Username string `tcp:"username"`
		TeamId   int    `tcp:"teamId"`
	}{"user:new", u.Id, u.Name, assignedTeamId}
	network.TcpClients.Broadcast <- msg

	helper.SendBadge("join", u.Id)

	// reply to sencha with user data
	return packet.Out{ "user:info", member.User }
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
	teams.Unregister <- e.Sender
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


func onCollectorHeartbeat(e events.Event) interface{} {
	// e.g. /name=collector:heartbeat/id=0/health=100/capacity=100/fill=0/color=#FFAA99$

	type collector struct {
		Id     		int	`tcp:"id"`
		Health 		int `tcp:"health"`
		Fill		int `tcp:"fill"`
		Capacity	int `tcp:"capacity"`
		Color		string `tcp:"color"`
	}
	c := collector{}
	tcp.Unmarshal(e.Args, &c)

	teamPrefix := fmt.Sprintf("team:%v:", c.Id)

	db.Redis.Set(teamPrefix + "health", strconv.Itoa(c.Health))
	db.Redis.Set(teamPrefix + "fill", strconv.Itoa(c.Fill))
	db.Redis.Set(teamPrefix + "capacity", strconv.Itoa(c.Capacity))
	db.Redis.Set(teamPrefix + "color", c.Color)

	return nil
}

func onCollectorMerge(e events.Event) interface{} {
	// e.g. /name=collector:merge/team_1=0/team_2=1$

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
		//Name   	string `tcp:"name"`
		Id     		int 	`tcp:"id"`
		Points 		int 	`tcp:"points"`
		Complete 	int		`tcp:"complete"`
	}
	c := collector{}
	tcp.Unmarshal(e.Args, &c)

	// give points!
	if team, ok := teams.Roster[c.Id]; ok {
		pts := c.Points / len(team)

		for _, member := range team {
			
			userHarvestKey := fmt.Sprintf("uid:%v:harvests", member.User.Id)

			if (c.Complete > 0) {
				helper.SendBadge("firstComplete", member.User.Id)
				rounds := db.Redis.Incr(userHarvestKey).Val()
				if (rounds > 3) {
					helper.SendBadge("bumperCrop", member.User.Id)
				}
			} else {
				db.Redis.Set(userHarvestKey, "0")
			}

			helper.SendPoints(pts, member.User.Id)
			userKey := fmt.Sprintf("uid:%v:points", member.User.Id)
			totalPts := db.Redis.IncrBy(userKey, int64(pts))

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

// Spit out performance statistics for entire program
func performanceHandler(w http.ResponseWriter, r *http.Request) {
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
