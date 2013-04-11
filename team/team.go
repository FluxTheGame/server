package team

import (
	"bitbucket.org/jahfer/flux-middleman/network"
	"bitbucket.org/jahfer/flux-middleman/helper"
	"bitbucket.org/jahfer/flux-middleman/packet"
	"bitbucket.org/jahfer/flux-middleman/user"
	"bitbucket.org/jahfer/flux-middleman/db"
	"encoding/json"
	"image/color"
	"strconv"
	"time"
	"math"
	"fmt"
	"io"
)

type Member struct {
	User user.User
	Conn io.Writer
}

type Manager struct {
	Roster 		map[int] []Member
	Queue 		chan Member
	Unregister 	chan io.Writer
	LastId	   	chan int
}

func NewManager() Manager {
	return Manager{
		Roster: make(map[int] []Member),
		Queue: make(chan Member),
		Unregister: make(chan io.Writer),
		LastId: make(chan int),
	}
}

func (t Manager) NumUsers() (count int) {
	for _, m := range t.Roster {
		count += len(m)
	}
	return
}

func (t Manager) MaxTeams() int {
	userCount := float64(t.NumUsers())
	max := math.Ceil( math.Sqrt(userCount) )
	
	if (max < 1) { 
		return 1 
	}

	return int(max)
}

func (t Manager) GetIndex(conn io.Writer) (int, int, int) {
	for teamId, team := range t.Roster {
		// for all members
		for userIndex, member := range team {
			// found disconnected member
			if member.Conn == conn {
				return teamId, member.User.Id, userIndex
			}
		}
	}

	// user not found
	return -1, -1, -1
}

func (t Manager) GetUserIndex(teamId, userId int) int {
	for i, member := range t.Roster[teamId] {
		if member.User.Id == userId {
			return i
		}
	}

	return -1
}

func (t *Manager) ReturnToQueue(teamId int) {

	members := t.Roster[teamId][0:]
	t.removeTeam(teamId)

	for _, member := range members {
		t.Queue <- member
		newTeamId := <-t.LastId
		t.memberChangeTeam(member.User.Id, newTeamId, newTeamId)
	}
}

func (t *Manager) removeTeam(teamId int) {
	teamKey := fmt.Sprintf("team:%v:users", teamId)
	db.Redis.Del(teamKey)
	delete(t.Roster, teamId)
	helper.ToXna("collector:destroy", teamId)
}

func (t *Manager) removeAnonConnection(conn io.Writer) {
	teamId, userId, userIndex := t.GetIndex(conn)
	t.RemoveMember(teamId, userId, userIndex)
}

func (t Manager) memberChangeTeam(userId, newTeamId, curTeamId int) {
	msg := struct {
		Name string `tcp:"name"`
		Id   int    `tcp:"id"`
		TeamId   int    `tcp:"teamId"`
	}{"user:newTeam", userId, newTeamId}

	network.TcpClients.Broadcast <- msg


	index := t.GetUserIndex(curTeamId, userId)

	toApp := packet.Out{
		Name:    "user:newTeam",
		Message: newTeamId,
	}

	out, _ := json.Marshal(toApp)
	t.Roster[curTeamId][index].Conn.Write(out)
}

func (t *Manager) RemoveMember(teamId, userId, userIndex int) {
	if teamId != -1 {

		uName := t.Roster[teamId][userIndex].User.Name
		
		t.Roster[teamId][userIndex] = t.Roster[teamId][len(t.Roster[teamId])-1]
		t.Roster[teamId] = t.Roster[teamId][0:len(t.Roster[teamId])-1]

		t.removeMemberKeys(userId)
		t.removeMemberFromTeam(userId, teamId)

		userIdKey := fmt.Sprintf("username:%v:uid", uName)
		db.Redis.Del(userIdKey)

		helper.ToXna("user:disconnect", userId)
	}
}

func (t Manager) removeMemberKeys(userId int) {
	// remove user from redis
	db.Redis.ZRem("global:clients", strconv.Itoa(userId))

	uidPrefix := fmt.Sprintf("uid:%v", userId)
	db.Redis.Del(uidPrefix + ":points")
	db.Redis.Del(uidPrefix + ":team")
	db.Redis.Del(uidPrefix + ":badges")
	db.Redis.Del(uidPrefix + ":username")
	db.Redis.Del(uidPrefix + ":shotsFired")
	db.Redis.Del(uidPrefix + ":harvests")
}

func (t *Manager) removeMemberFromTeam(userId, teamId int) {
	// delete user from team
	teamKey := fmt.Sprintf("team:%v:users", teamId)
	db.Redis.SRem(teamKey, strconv.Itoa(userId))

	// remove team if empty
	if len(t.Roster[teamId]) < 1 {
		t.removeTeam(teamId)
	}
}

func (t *Manager) addMember(m Member) (teamId int, err error) {

	if len(t.Roster) < t.MaxTeams() {
		teamId = t.createNewTeam()
		t.Roster[teamId] = []Member{ m }
	} else {
		smallest := t.getSmallestTeam()
		// add users to existing team
		t.Roster[smallest] = append(t.Roster[smallest], m)
		teamId = smallest
	}

	if len(t.Roster[teamId]) >= 8 {
		for _, m := range t.Roster[teamId] {
			helper.SendBadge("theOcho", m.User.Id)
		}
	}

	m.User.TeamId = teamId

	// add user to team list in DB
	key := fmt.Sprintf("team:%v:users", teamId)
	db.Redis.SAdd(key, strconv.Itoa(m.User.Id))

	key = fmt.Sprintf("uid:%v:team", m.User.Id)
	db.Redis.Set(key, strconv.Itoa(teamId))

	return
}

func (t *Manager) CheckExpired() {
	expired := db.Redis.ZRangeByScore("global:clients", "-inf", strconv.FormatInt(time.Now().Unix()-10, 10), 0, -1).Val()
	if len(expired) > 0 {
		for _, idStr := range expired {
			db.Redis.ZRem("global:clients", idStr)
			id, _ := strconv.Atoi(idStr)
			teamId, _ := strconv.Atoi(db.Redis.Get("uid:" + idStr + ":team").Val())
			userIndex := t.GetUserIndex(teamId, id)
			if userIndex == -1 {
				fmt.Printf("[ERROR]\tUser's index out of bounds\n")
				continue
			}
			t.RemoveMember(teamId, id, userIndex)
		}
	}
}

func (t Manager) Analytics() {
	for _, team := range t.Roster {
		for _, member := range team {
			userShotKey := fmt.Sprintf("uid:%v:shotsFired", member.User.Id)
			shotsFired, _ := strconv.Atoi(db.Redis.Get(userShotKey).Val())
			if shotsFired > 2 {
				helper.SendBadge("triggerHappy", member.User.Id)
			}
			db.Redis.Del(userShotKey)
		}
	}
}

func (t *Manager) getSmallestTeam() int {
	smallestSize := 10000
	var smallestIndex int

	for key, team := range t.Roster {
		if len(team) < smallestSize {
			smallestSize  = len(team)
			smallestIndex = key
		}
	}

	return smallestIndex
}

func (t *Manager) createNewTeam() int {

	defer db.Redis.Incr("global:nextTeamId")

	get := db.Redis.Get("global:nextTeamId")
	teamId, _ := strconv.Atoi(get.Val())

	c := GetNextColor()

	msg := struct {
		Name string `tcp:"name"`
		Id   int    `tcp:"id"`
		Color color.Color `tcp:"color"`
	}{"collector:new", teamId, c}

	network.TcpClients.Broadcast <- msg

	return teamId
}


// event objects
type Merger struct {
	TeamId1 int `tcp:"team_1"`
	TeamId2 int `tcp:"team_2"`
}

func (t *Manager) Merge(teams Merger) {

	team1 	:= fmt.Sprintf("team:%v:users", teams.TeamId1)
	team2 	:= fmt.Sprintf("team:%v:users", teams.TeamId2)

	// delete members
	defer db.Redis.Del(team2)
	defer delete(t.Roster, teams.TeamId2)

	db.Redis.SUnionStore(team1, team1, team2)

	for _, idStr := range db.Redis.SMembers(team2).Val() {
		userId, _ := strconv.Atoi(idStr)
		// update user id
		teamKey := fmt.Sprintf("uid:%v:team", userId)
		db.Redis.Set(teamKey, strconv.Itoa(teams.TeamId1))
		// tell xna new team id
		t.memberChangeTeam(userId, teams.TeamId1, teams.TeamId2)
	}


	// move members to team 1
	t.Roster[teams.TeamId1] = append(t.Roster[teams.TeamId1], t.Roster[teams.TeamId2]...)

	// everybody, celebrate merge!
	for _, idStr := range db.Redis.SMembers(team1).Val() {
		userId, _ := strconv.Atoi(idStr)
		helper.SendBadge("firstMerge", userId)

		if len(t.Roster[teams.TeamId1]) >= 8 {
			helper.SendBadge("theOcho", userId)
		}
	}
}

// Boot cycle for team manager
func (t *Manager) Run() {
	for {
		select {
		// add new client
		case member := <-t.Queue:
			teamId, _ := t.addMember(member)
			t.LastId <- teamId
		// user has disconnected
		case deadClient := <-t.Unregister:
			go t.removeAnonConnection(deadClient)
		}
	}
}