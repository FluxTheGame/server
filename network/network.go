package network

import (
	"bitbucket.org/jahfer/flux-middleman/client"
	"bitbucket.org/jahfer/flux-middleman/events"
	"bitbucket.org/jahfer/flux-middleman/db"
	"code.google.com/p/go.net/websocket"
	r "github.com/vmihailenco/redis"
	"net"
	"net/http"
	"fmt"
	"os"
)

// store all client connections
var WsClients 	= client.NewHub() // sencha users
var TcpClients 	= client.NewHub() // xna
// Create event manager for dispatches
var Manager 	= events.NewManager()

var globalInit = make(chan bool, 3)

// Boot cycle for servers
func Init() {
	fmt.Println(" Welcome to Flux!")
	fmt.Println("-----------------------------------------------")
	fmt.Println(" Booting up background services...")

	go initTcpServer()
	go initSocketServer()

	go initDb()
	defer db.Close()

	count := 0

	go func() {
		for _ = range globalInit {
			count++;
			if count >= 3 {
				fmt.Println(" Initialization complete!")
				fmt.Println("")
			}
		}
	}()

	Manager.Listener()
}

// Called on every new WebSocket connection
func wsHandler(ws *websocket.Conn) {
	// create client object
	c := &client.WebSocketClient{Conn: ws}
	c.Send = make(chan []byte, 256)

	// register client in list, and boot up to read/write
	WsClients.Register <- c
	defer func() { WsClients.Unregister <- c }()
	go c.Sender()
	c.Listener(Manager.Incoming)
}

// Called on every new TCP connection
func tcpHandler(conn net.Conn) {
	// create client object
	c := &client.TcpClient{Conn: conn}
	c.Send = make(chan []byte, 256)

	// register client in list, and boot up to read/write
	TcpClients.Register <- c
	defer func() { TcpClients.Unregister <- c }()
	go c.Sender()
	c.Listener(Manager.Incoming)
}

// Start the HTTP/WS server to listen for new connections
func initSocketServer() {
	fmt.Println(" -- Initializing WebSocket server on :80")

	go WsClients.Run()

	currentDirectory, _ := os.Getwd()

	// serve api
	http.HandleFunc("/api/v1/Collector.json", handleApiCollector)
	http.HandleFunc("/api/v1/Badges.json", handleApiBadges)
	// endpoint for websocket connections
	http.Handle("/ws", websocket.Handler(wsHandler))
	// serve static files for Sencha
	http.Handle("/", http.FileServer(http.Dir(currentDirectory + "/GameController")))

	globalInit <- true

	if err := http.ListenAndServe(":80", nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

// Start the TCP server and listen for new connections
func initTcpServer() {
	fmt.Println(" -- Initializing TCP server on :8100")

	go TcpClients.Run()

	listener, err := net.Listen("tcp", ":8100")
	if err != nil {
		panic(err.Error())
	}

	defer listener.Close()

	globalInit <- true

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err.Error())
		}
		go tcpHandler(conn)
	}
}

func initDb() {
	fmt.Println(" -- Initializing Redis server on :6379")

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ERROR]\tRedis database connection not found.\n")
			os.Exit(1)
		}
	}()

	db.Init()

	_, err := db.Redis.Pipelined(func(c *r.PipelineClient) {
	    db.Redis.Set("global:nextUserId", "0")
	    db.Redis.Set("global:nextTeamId", "0")
	})

	if err != nil {
		fmt.Printf("[ERROR]\tCould not write to Redis database.\n")
		panic(err)
	}

	globalInit <- true
}
