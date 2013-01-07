package main

import (
	"net"
	"net/http"
	"fmt"
	"time"
	"code.google.com/p/go.net/websocket"
)

var wsClients  = NewHub()
var tcpClients = NewHub()
var events = NewEventManager()

func onUserBloat(user UserId) {
	fmt.Println("[Bloat] User:", user)
}

func wsHandler(ws *websocket.Conn) {
	fmt.Println("-- client connected")

	c := &webSocketClient{Conn: ws}
	c.Send = make(chan []byte, 256)

	wsClients.register <- c
	defer func() { wsClients.unregister <- c }()
	go c.sender()
	c.listener()
}

func tcpHandler(conn net.Conn) {
	c := &tcpClient{Conn: conn}
	c.Send = make(chan []byte, 256)

	tcpClients.register <- c
	defer func() { tcpClients.unregister <- c }()
	go c.sender()
	c.listener()
}

func main() {

	events.HandleFunc("user:bloat", makeEventHandler(onUserBloat))

	go initTcpServer()
	go initSocketServer()
	go events.listen()

	for {
		msg := OutPacket{
			Name: "server:heartbeat",
			Message: "Heartbeat " + time.Now().Format(time.StampMilli),
		}
		wsClients.broadcast  <- msg
		tcpClients.broadcast <- msg
		time.Sleep(1000 * time.Millisecond)
	}
}

func initSocketServer() {
	fmt.Println("-- Initializing WS server on :8080")

	go wsClients.run()

	http.Handle("/", websocket.Handler(wsHandler))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func initTcpServer() {
	fmt.Println("-- Initializing TCP server on :8100")

	go tcpClients.run()

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