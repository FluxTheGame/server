package network

import (
	"bitbucket.org/jahfer/flux-middleman/client"
	"bitbucket.org/jahfer/flux-middleman/events"
	"code.google.com/p/go.net/websocket"
	"net"
	"net/http"
	_ "time"
	"fmt"
)

// store all client connections
var WsClients 	= client.NewHub() // sencha users
var TcpClients 	= client.NewHub() // xna
// Create event manager for dispatches
var Manager 	= events.NewManager()

// Boot cycle for servers
func Init() {
	go initTcpServer()
	go initSocketServer()
	Manager.Listener()

	/*for {
		time.Sleep(1000 * time.Millisecond)
	}*/
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

// Start the TCP server and listen for new connections
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