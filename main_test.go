package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"net"
	"testing"
	"time"
)

var startedMain bool = false

func BenchmarkWebSocketConnection(b *testing.B) {
	b.StopTimer()

	if !startedMain {
		go main()
		startedMain = true
		time.Sleep(100)
	}

	srvAddr := "localhost:8080"
	config, _ := wsConnSetup(srvAddr)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		client, _ := net.Dial("tcp", srvAddr)
		conn, _ := websocket.NewClient(config, client)

		msg := []byte(`[{
			"name": "user:new", 
			"args": {"id": -1}
		}]`)

		conn.Write(msg)
		conn.Close()
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
