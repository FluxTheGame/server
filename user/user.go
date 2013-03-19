package user

import (
	"bitbucket.org/jahfer/flux-middleman/db"
	"strconv"
	"fmt"
)

type Id struct {
	Id int `json:"id"`
	TeamId int `json:"team_id"`
}

type User struct {
	Name 	string 	`json:"name"`
	Id  	int    	`json:"id"`
	TeamId  int 	`json:"team_id"`
	Points  int 	`json:"points"`
}

func (u *User) Save() (error, error) {
	// set ID for user
	u.Id = getNextId()
	// store user in DB
	key := fmt.Sprintf("username:%v:uid", u.Name)
	setId := db.Redis.Set(key, strconv.Itoa(u.Id))
	usernameKey := fmt.Sprintf("uid:%v:username", u.Id)
	setName := db.Redis.Set(usernameKey, u.Name)
	
	return setId.Err(), setName.Err()
}

type Coords struct {
	Id int `json:"id"`
	X  int `json:"x"`
	Y  int `json:"y"`
}

func getNextId() (id int) {
	get := db.Redis.Get("global:nextUserId")
	if err := get.Err(); err != nil {
		panic("Could not get next user id " + err.Error())
	}

	defer db.Redis.Incr("global:nextUserId")

	val, _ := strconv.ParseInt(get.Val(), 10, 0)
	id = int(val)

	return
}