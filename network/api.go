package network

import (
	"bitbucket.org/jahfer/flux-middleman/db"
	"encoding/json"
	"net/http"
	"fmt"
)

func handleApiCollector(w http.ResponseWriter, r *http.Request) {
	out := json.NewEncoder(w)

	teamId := r.FormValue("id")
	teamCheck := db.Redis.SCard("team:" + teamId + ":users")
	if err := teamCheck.Err(); err != nil || teamCheck.Val() == 0 {
		out.Encode(struct {
			Error string
		}{"Team not found: " + teamId})
		return
	}

	teamPrefix := fmt.Sprintf("team:%v:", teamId)

	health 		:= db.Redis.Get(teamPrefix + "health")
	fill 		:= db.Redis.Get(teamPrefix + "fill")
	capacity 	:= db.Redis.Get(teamPrefix + "capacity")
	teamMembers := db.Redis.SMembers(teamPrefix + "users")

	var users []string
	for _, id := range teamMembers.Val() {
		userIdKey := fmt.Sprintf("uid:%v:username", id)
		fmt.Printf("[API:USERS]\t%v\n", id)
		users = append(users, db.Redis.Get(userIdKey).Val())
	}
	
	fmt.Printf("[API:USERNAMES]\t%v\n", users)

	col := struct {
		Health 		string 		`json:"health"`
		Fill 		string 		`json:"fill"`
		Capacity 	string 		`json:"cap"`
		Team 		[]string 	`json:"team"`
		Id 			string 		`json:"id"`
	}{health.Val(), fill.Val(), capacity.Val(), users, teamId}

	obj := struct {
		Collector interface{} `json:"collector"`
	}{col}

	out.Encode(obj)
}

func handleApiBadges(w http.ResponseWriter, r *http.Request) {
	out := json.NewEncoder(w)

	userId := r.FormValue("id")
	badges := fmt.Sprintf("uid:%v:badges", userId)
	badgeSet := db.Redis.SMembers(badges)

	badgeNames := badgeSet.Val()

	obj := struct {
		UserId string `json:"id"`
		Badges []string `json:"badges"`
	}{userId, badgeNames}

	out.Encode(obj)
}