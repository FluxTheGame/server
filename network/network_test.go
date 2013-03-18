package network

import (
	"testing"
	"net"
	"code.google.com/p/go.net/websocket"
	"fmt"
)

func TestTcpConnection(t *testing.T) {
	go Init()

	srvAddr := "localhost:8100"
	tcpAddr, err := net.ResolveTCPAddr("tcp", srvAddr)
	if err != nil {
		t.Errorf("Could not find TCP address:", err.Error())
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		t.Errorf("Could not connect to TCP server:", err.Error())
	}

	 _, err = conn.Write([]byte("This is a test"))
	if err != nil {
		t.Errorf("Write to server failed:", err.Error())
	}

}

func wsConnSetup(srvAddr string) (config *websocket.Config, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", srvAddr)
	if err != nil {
		return nil, err
	}

	config, _ = websocket.NewConfig(fmt.Sprintf("ws://%s%s", tcpAddr, "/ws"), "http://localhost/ws")

	return
}

func TestWebSocketConnection(t *testing.T) {

	srvAddr := "localhost:8080"
	config, err := wsConnSetup(srvAddr)
	if err != nil {
		t.Errorf("Could not find TCP address:", err.Error())
	}

	client, err := net.Dial("tcp", srvAddr)
	if err != nil {
		t.Fatal("dialing", err)
	}

	conn, err := websocket.NewClient(config, client)
	if err != nil {
		t.Errorf("WebSocket handshake error: %v", err)
		return
	}

	//msg := []byte("hello, world\n")
	msg := []byte(`[{
		"name": "user:join", 
		"args": {"id": -1}
	}]`)

	if _, err := conn.Write(msg); err != nil {
		t.Errorf("Write: %v", err)
	}

	conn.Close()

}

func BenchmarkWebSocketConnection(b *testing.B) {

	srvAddr := "localhost:8080"
	config, _ := wsConnSetup(srvAddr)

	for i:=0; i<b.N; i++ {

		client, _ := net.Dial("tcp", srvAddr)

		conn, _ := websocket.NewClient(config, client)

		//msg := []byte("hello, world\n")
		msg := []byte(`[{
			"name": "user:join", 
			"args": {"id": -1}
		}]`)

		conn.Write(msg)

		conn.Close()
	}
}