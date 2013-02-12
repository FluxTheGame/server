package db

import (
	"fmt"
	r "github.com/vmihailenco/redis"
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/native"
	"net"
)

var Redis *r.Client

func Init() {
	srvAddr := "localhost:6379"

	Redis = r.NewTCPClient(srvAddr, "", -1)

	tcpAddr, _ := net.ResolveTCPAddr("tcp", srvAddr)
	if _, err := net.DialTCP("tcp", nil, tcpAddr); err != nil {
		panic(err)
	}
}

func Close() {
	Redis.Close();
}

func Examples() {

	/* == CONNECT MYSQL DATABASE == */
	mysqluser := "root"
	mysqlpass := "root"
	mysqldb := "flux"

	db := mysql.New("tcp", "", "127.0.0.1:8889", mysqluser, mysqlpass, mysqldb)

	err := db.Connect()
	if err != nil {
		panic(err.Error())
	}

	/* == TEST MYSQL == */
	// submit a new user
	/*u := &user.User{Name: "John Doe", Id: 10}
	newUser, err := db.Prepare("INSERT INTO users (name) VALUES (?)")
	_, err = newUser.Run(u.Name)
	if err != nil {
		panic(err.Error())
	}*/

	// get all users
	rows, res, err := db.Query("SELECT * FROM users")
	if err != nil {
		panic(err.Error())
	}

	for _, row := range rows {
		for _, col := range row {
			if col == nil {
				// ...
			} else {
				//fmt.Println(col)
			}
		}

		id := res.Map("id")
		name := res.Map("name")

		fmt.Println("Data collected:", row.Int(id), row.Str(name))

	}
}
