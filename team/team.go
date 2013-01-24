package team

import (
	"bitbucket.org/jahfer/flux-middleman/user"
	"bitbucket.org/jahfer/flux-middleman/db"
	"strconv"
	"math"
	"fmt"
	"io"
)

type Member struct {
	User user.User
	Conn io.Writer
}

type Manager struct {
	Roster [/*team id*/][/*user index*/]Member
	Queue chan Member
	Unregister chan io.Writer
}

func NewManager() Manager {
	return Manager{
		Roster: make([][]Member, 1),
		Queue: make(chan Member),
		Unregister: make(chan io.Writer),
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

// Boot cycle for team manager
func (t *Manager) Run() {

	t.Roster[0] = []Member{}

	for {
		select {
		// add new client
		case member := <-t.Queue:
			t.addMember(member)
		// user has disconnected
		case deadClient := <-t.Unregister:
			go t.removeMember(deadClient)
		}
	}
}

func (t *Manager) GetIndex(conn io.Writer) (int, int) {
	for i, team := range t.Roster {
		// for all members
		for j, m := range team {
			// found disconnected member
			if m.Conn == conn {
				return i, j
			}
		}
	}

	// user not found
	return -1, -1
}

func (t *Manager) removeMember(conn io.Writer) {

	teamId, id := t.GetIndex(conn)

	if teamId != -1 {
		// swap index to delete with last element, then cut off last item
		// ** does not maintain order!
		t.Roster[teamId][id] = t.Roster[teamId][len(t.Roster[teamId])-1]
		t.Roster[teamId] = t.Roster[teamId][0:len(t.Roster[teamId])-1]

		teamKey := fmt.Sprintf("team:%v:users", teamId)
		//fmt.Printf("Removing user %v from team %v\n", id, teamId)
		//db.Redis.LRem(teamKey, 0, strconv.Itoa(id))

		// if team is now empty...and not the last team!
		if len(t.Roster[teamId]) < 1 && len(t.Roster) > 1 {
			// remove!
			t.Roster[teamId] = t.Roster[len(t.Roster)-1]
			t.Roster = t.Roster[0:len(t.Roster)-1]

			db.Redis.Del(teamKey)
		}
	}
}

func (t *Manager) addMember(m Member) {

	var teamId int

	if len(t.Roster) < t.MaxTeams() {
		// create a new collector
		t.Roster = append(t.Roster, []Member{m})
		teamId = len(t.Roster)-1

	} else {
		// add users to existing collection
		smallest := 0
		for i, team := range t.Roster {
			if len(team) < len(t.Roster[smallest]) {
				smallest = i
			}
		}
		t.Roster[smallest] = append(t.Roster[smallest], m)
		teamId = smallest
	}

	// add user to team list in DB
	key := fmt.Sprintf("team:%v:users", teamId)
	push := db.Redis.RPush(key, strconv.Itoa(m.User.Id))
	if err := push.Err(); err != nil {
		panic(err)
	}

	key = fmt.Sprintf("uid:%v:team", m.User.Id)
	set := db.Redis.Set(key, strconv.Itoa(teamId))
	if err := set.Err(); err != nil {
		panic(err)
	}

}