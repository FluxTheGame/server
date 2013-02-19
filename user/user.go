package user

import (
	"bitbucket.org/jahfer/flux-middleman/db"
	"strconv"
	"fmt"
)

type Id struct {
	Id int `json:"id"`
}

type User struct {
	Name 	string 	`json:"name"`
	Id  	int    	`json:"id"`
	TeamId  int 	`json:"team_id"`
}

func (u *User) Save() error {
	// set ID for user
	u.Id = getNextId()
	// store user in DB
	key := fmt.Sprintf("username:%v:uid", u.Name)
	set := db.Redis.Set(key, strconv.Itoa(u.Id))
	
	return set.Err()
}

type Coords struct {
	Id int `json:"id"`
	X  int `json:"x"`
	Y  int `json:"y"`
}

func getNextId() (id int) {
	get := db.Redis.Get("global:nextUserId")
	if err := get.Err(); err != nil {
		panic(err)
	}

	defer db.Redis.Incr("global:nextUserId")

	val, _ := strconv.ParseInt(get.Val(), 10, 0)
	id = int(val)

	return
}