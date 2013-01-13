package team

import (
	"math"
	"fmt"
	"io"
)

var lastId int = 0

func LastId() int {
	id := lastId
	lastId++
	return id
}

type Member struct {
	Id int
	teamId int
	Conn io.Writer
}

type Manager struct {
	Roster [][]Member // list of team ids and members per team
	Queue chan Member // user ids
	Unregister chan io.Writer
}

func NewManager() Manager {
	return Manager{
		Roster: make([][]Member, 1),
		Queue: make(chan Member),
		Unregister: make(chan io.Writer),
	}
}


// Shows number of open connections, not logged-in users
func (t Manager) NumUsers() int {
	count := 0

	for _, m := range t.Roster {
		count += len(m)
	}

	return count
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
			// register client to team
			t.add(member)
		
		// user has disconnected
		case deadClient := <-t.Unregister:
			// for all teams
			for i, team := range t.Roster {
				// for all members
				for j, m := range team {
					// if this is the disconnected member
					if m.Conn == deadClient {
						// move last elem to i-th position
						// to replace dead index
  						fmt.Println("Before:", t.Roster)
						team[j] = team[len(team)-1]
  						t.Roster[i] = team[0:len(team)-1]
  						fmt.Println("After: ", t.Roster)

  						// if team is now empty...
  						if len(t.Roster[i]) < 1 {
  							// remove!
  							t.Roster[i] = t.Roster[len(t.Roster)-1]
							t.Roster = t.Roster[0:len(t.Roster)-1]
  						}

						break
					}
				}
			}
		}
	}
}

func (t *Manager) add(m Member) {

	if len(t.Roster) < t.MaxTeams() {
		// create a new collector
		m.teamId = len(t.Roster)
		t.Roster = append(t.Roster, []Member{m})
	} else {
		// add users to existing collection
		smallest := 0
		for i, team := range t.Roster {
			if len(team) < len(t.Roster[smallest]) {
				smallest = i
			}
		}
		m.teamId = smallest
		t.Roster[smallest] = append(t.Roster[smallest], m)
	}

}