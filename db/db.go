package db

import (
	"fmt"
	r "github.com/vmihailenco/redis"
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/native"
	"net"
	"time"
	"os/exec"
)

var Redis *r.Client

func Init() {
	srvAddr := "localhost:6379"

	Redis = r.NewTCPClient(srvAddr, "", -1)

	tcpAddr, _ := net.ResolveTCPAddr("tcp", srvAddr)

	if _, err := net.DialTCP("tcp", nil, tcpAddr); err != nil {
		fmt.Printf(" [NOTICE]\tRedis server not connected.\n")
		fmt.Printf(" [NOTICE]\tAttempting to boot up Redis server.\n")

		response := make(chan bool, 1)

		go bootRedisServer(response);

		<-response
		fmt.Printf(" [NOTICE]\tRedis server has connected.\n")
	}
	
	// clear stale data
	Redis.FlushDb()
}

func bootRedisServer(resp chan bool) {
	cmd := exec.Command("redis-server")

	if err := cmd.Start(); err != nil {
		fmt.Printf(" [ERROR]\tCould not connect to Redis server.\n")
		fmt.Println(err.Error())
		panic(1)
	}

	time.Sleep(50 * time.Millisecond)

	resp <- true

	if err := cmd.Wait(); err != nil {
		fmt.Printf(" [ERROR]\tRedis server disconnected.\n")
		panic(2);
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
