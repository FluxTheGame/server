package team

import (
	"bitbucket.org/jahfer/flux-middleman/network"
	"bitbucket.org/jahfer/flux-middleman/helper"
	"bitbucket.org/jahfer/flux-middleman/user"
	"bitbucket.org/jahfer/flux-middleman/db"
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
		fmt.Println("Returning to queue...")
		// add user to team list in DB
		key := fmt.Sprintf("team:%v:users", teamId)
		db.Redis.SRem(key, strconv.Itoa(member.User.Id))

		t.Queue <- member

		// get team id
		assignedTeamId := <-t.LastId
		member.User.TeamId = assignedTeamId
	}
}

func (t *Manager) removeAnonConnection(conn io.Writer) {
	teamId, userId, userIndex := t.GetIndex(conn)
	t.RemoveMember(teamId, userId, userIndex)
}

func (t *Manager) RemoveMember(teamId, userId, userIndex int) {
	if teamId != -1 {
		// delete user
		defer func () {
			t.Roster[teamId][userIndex] = t.Roster[teamId][len(t.Roster[teamId])-1]
			t.Roster[teamId] = t.Roster[teamId][0:len(t.Roster[teamId])-1]
		}()

		t.removeMemberKeys(userId)

		userIdKey := fmt.Sprintf("username:%v:uid", t.Roster[teamId][userIndex].User.Name)
		db.Redis.Del(userIdKey)

		t.removeMemberFromTeam(userId, teamId)

		helper.ToXna("user:disconnect", userId)
	}
}

func (t *Manager) removeTeam(teamId int) {
	teamKey := fmt.Sprintf("team:%v:users", teamId)
	delete(t.Roster, teamId)
	db.Redis.Del(teamKey)
	helper.ToXna("collector:destroy", teamId)
}

func (t Manager) removeMemberKeys(userId int) {
	// remove user from redis
	db.Redis.ZRem("global:clients", strconv.Itoa(userId))

	uidPrefix := fmt.Sprintf("uid:%v", userId)
	db.Redis.Del(uidPrefix + ":points")
	db.Redis.Del(uidPrefix + ":team")
	db.Redis.Del(uidPrefix + ":badges")
	db.Redis.Del(uidPrefix + ":username")
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
	if err := get.Err(); err != nil {
		panic(err)
	}
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
	// delete members
	defer delete(t.Roster, teams.TeamId2)

	team1 	:= fmt.Sprintf("team:%v:users", teams.TeamId1)
	team2 	:= fmt.Sprintf("team:%v:users", teams.TeamId2)
	defer db.Redis.Del(team2)

	db.Redis.SUnionStore(team1, team1, team2)

	for _, usr := range db.Redis.SMembers(team1).Val() {
		teamKey := fmt.Sprintf("uid:%v:team", usr)
		db.Redis.Set(teamKey, strconv.Itoa(teams.TeamId1))

		id, _ := strconv.Atoi(usr)
		helper.SendBadge("firstMerge", id)
	}

	// move members to new team
	t.Roster[teams.TeamId1] = append(t.Roster[teams.TeamId1], t.Roster[teams.TeamId2]...)
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