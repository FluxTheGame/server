package team

import (
	"bitbucket.org/jahfer/flux-middleman/network"
	"bitbucket.org/jahfer/flux-middleman/user"
	"bitbucket.org/jahfer/flux-middleman/db"
	"image/color"
	"strconv"
	"math"
	"fmt"
	"io"
)

// event objects
type Merger struct {
	TeamId1 int `tcp:"team_1"`
	TeamId2 int `tcp:"team_2"`
}

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

func (t *Manager) Merge(teams Merger) {
	// delete members
	defer delete(t.Roster, teams.TeamId1)
	defer delete(t.Roster, teams.TeamId2)

	newTeamId := t.createNewTeam()

	dest 	:= fmt.Sprintf("team:%v:users", newTeamId)
	team1 	:= fmt.Sprintf("team:%v:users", teams.TeamId1)
	team2 	:= fmt.Sprintf("team:%v:users", teams.TeamId2)

	db.Redis.SUnionStore(dest, team1, team2)
	db.Redis.Del(team1, team2)

	for _, usr := range db.Redis.SMembers(dest).Val() {
		teamKey := fmt.Sprintf("uid:%v:team", usr)
		db.Redis.Set(teamKey, strconv.Itoa(newTeamId))

		badgesKey := fmt.Sprintf("uid:%v:badges", usr)
		db.Redis.SAdd(badgesKey, "firstMerge")
	}

	// move members to new team
	t.Roster[newTeamId] = append(t.Roster[teams.TeamId1], t.Roster[teams.TeamId2]...)
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

func (t *Manager) GetIndex(conn io.Writer) (int, int, int) {
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

func (t *Manager) removeMember(conn io.Writer) {

	teamId, userId, userIndex := t.GetIndex(conn)

	if teamId != -1 {
		// remove user from redis
		userPointsKey := fmt.Sprintf("uid:%v:points", userId)
		db.Redis.Del(userPointsKey)
		userTeamKey := fmt.Sprintf("uid:%v:team", userId)
		db.Redis.Del(userTeamKey)
		userIdKey := fmt.Sprintf("username:%v:uid", t.Roster[teamId][userIndex].User.Name)
		db.Redis.Del(userIdKey)

		// delete user
		t.Roster[teamId][userIndex] = t.Roster[teamId][len(t.Roster[teamId])-1]
		t.Roster[teamId] = t.Roster[teamId][0:len(t.Roster[teamId])-1]
		
		// delete user from team
		teamKey := fmt.Sprintf("team:%v:users", teamId)
		db.Redis.SRem(teamKey, strconv.Itoa(userId))

		// remove team if empty
		if len(t.Roster[teamId]) < 1 && len(t.Roster) > 1 {
			delete(t.Roster, teamId)
			db.Redis.Del(teamKey)
		}
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
	val, _ := strconv.ParseInt(get.Val(), 10, 0)
	teamId := int(val)


	c := GetNextColor()

	msg := struct {
		Name string `tcp:"name"`
		Id   int    `tcp:"id"`
		Color color.Color `tcp:"color"`
	}{"collector:new", teamId, c}

	network.TcpClients.Broadcast <- msg

	return teamId
}



// Boot cycle for team manager
func (t *Manager) Run() {

	t.Roster[0] = []Member{}

	for {
		select {
		// add new client
		case member := <-t.Queue:
			teamId, _ := t.addMember(member)
			t.LastId <- teamId
		// user has disconnected
		case deadClient := <-t.Unregister:
			go t.removeMember(deadClient)
		}
	}
}