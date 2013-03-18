package network

import (
	"bitbucket.org/jahfer/flux-middleman/db"
	"encoding/json"
	"net/http"
	"fmt"
)

func handleApiCollector(w http.ResponseWriter, r *http.Request) {
	out := json.NewEncoder(w)

	userId := r.FormValue("id")
	teamId := db.Redis.Get("uid:" + userId + ":team")
	if err := teamId.Err(); err != nil {
		out.Encode(struct {
			Error string
		}{"User or team not found."})
		return
	}

	teamPrefix := fmt.Sprintf("team:%v:", teamId.Val())

	health := db.Redis.Get(teamPrefix + "health")
	fill := db.Redis.Get(teamPrefix + "fill")
	capacity := db.Redis.Get(teamPrefix + "capacity")
	teamMembers := db.Redis.SMembers(teamPrefix + "users")

	var users []string
	for id := range teamMembers.Val() {
		userIdKey := fmt.Sprintf("uid:%v:username", id)
		users = append(users, db.Redis.Get(userIdKey).Val())
	}

	col := struct {
		Health string `json:"health"`
		Fill string `json:"fill"`
		Capacity string `json:"cap"`
		Team []string `json:"team"`
	}{health.Val(), fill.Val(), capacity.Val(), users}

	obj := struct {
		UserId string `json:"id"`
		Collector interface{} `json:"collector"`
	}{userId, col}

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