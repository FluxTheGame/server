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

func TestWebSocketConnection(t *testing.T) {

	srvAddr := "localhost:8080"
	tcpAddr, err := net.ResolveTCPAddr("tcp", srvAddr)
	if err != nil {
		t.Errorf("Could not find TCP address:", err.Error())
	}

	config, _ := websocket.NewConfig(fmt.Sprintf("ws://%s%s", tcpAddr, "/"), "http://localhost")

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
		"args": {"id": 3, "x": 13, "y": 306}
	}]`)

	if _, err := conn.Write(msg); err != nil {
		t.Errorf("Write: %v", err)
	}

	/*var actual_msg = make([]byte, 512)
	bytesRead, err := conn.Read(actual_msg)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	actual_msg = actual_msg[0:bytesRead]
	var resp []packet.Out

	err = json.Unmarshal(actual_msg, &resp)
	if err != nil {
		t.Fatal("Packet sent from server not of type packet.Out", err.Error())
	}*/

	conn.Close()

}