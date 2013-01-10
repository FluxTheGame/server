package network

import (
	"bitbucket.org/jahfer/flux-middleman/client"
	"bitbucket.org/jahfer/flux-middleman/events"
	"bitbucket.org/jahfer/flux-middleman/packet"
	"bitbucket.org/jahfer/flux-middleman/team"
	"code.google.com/p/go.net/websocket"
	"net"
	"net/http"
	"time"
	"fmt"
)

var WsClients = team.NewHub()
var TcpClients = client.NewHub()
var Manager = events.NewManager()

func Init() {
	go initTcpServer()
	go initSocketServer()
	go Manager.Listener()

	for {
		msg := packet.Out{
			Name:    "server:heartbeat",
			Message: "Heartbeat " + time.Now().Format(time.StampMilli),
		}
		WsClients.Broadcast <- msg
		TcpClients.Broadcast <- msg
		time.Sleep(1000 * time.Millisecond)
		//fmt.Println(WsClients.NumUsers(), "users connected")
	}
}

func wsHandler(ws *websocket.Conn) {
	fmt.Println("-- ws client connected")

	c := &client.WebSocketClient{Conn: ws}
	c.Send = make(chan []byte, 256)

	WsClients.Register <- c
	defer func() { WsClients.Unregister <- c }()
	go c.Sender()
	c.Listener(Manager.Incoming)
}

func tcpHandler(conn net.Conn) {
	fmt.Println("-- tcp client connected")

	c := &client.TcpClient{Conn: conn}
	c.Send = make(chan []byte, 256)

	TcpClients.Register <- c
	defer func() { TcpClients.Unregister <- c }()
	go c.Sender()
	c.Listener(Manager.Incoming)
}

func initSocketServer() {
	fmt.Println("-- Initializing WS server on :8080")

	go WsClients.Run()

	http.Handle("/", websocket.Handler(wsHandler))
	http.HandleFunc("/wstest", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func initTcpServer() {
	fmt.Println("-- Initializing TCP server on :8100")

	go TcpClients.Run()

	listener, err := net.Listen("tcp", ":8100")
	if err != nil {
		panic(err.Error())
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err.Error())
		}
		go tcpHandler(conn)
	}
}