package user

type Id struct {
	Id int `json:"id"`
}

type User struct {
	Name string `json:"name"`
	Id   int    `json:"id"`
}

type Coords struct {
	Id int `json:"id"`
	X  int `json:"x"`
	Y  int `json:"y"`
}