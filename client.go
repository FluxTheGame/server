package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io"
	"net"
	"errors"
	"encoding/json"
	"reflect"
)


/* ===|==================================
 * Client Interface
 * ======================================*/
type client interface {
	io.WriteCloser
	Format(v interface{}) []byte
}


/* ===|==================================
 * Generic Client 
 * ======================================*/
type genericClient struct {
	Conn io.ReadWriteCloser
	Send chan []byte
}

func (c *genericClient) Write(b []byte) (n int, err error) {
	select {
	case c.Send <- b:
	default:
		return -1, errors.New("Write to client failed!")
	}

	return 0, nil
}

func (c *genericClient) Close() error {
	if c.Send != nil {
		close(c.Send)
	}
	return nil
}

func (c *genericClient) Format(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err.Error())
	}
	return data
}


/* ===|==================================
 * TCP Client 
 * ======================================*/
type tcpClient struct {
	genericClient
	Conn net.Conn
}

func (c *tcpClient) Format(v interface{}) []byte {
	ref := reflect.ValueOf(&v)
	return c.reflectValue(ref)
}

func (c *tcpClient) reflectValue(v reflect.Value) []byte {

	kind := v.Type()

	switch v.Kind() {
	case reflect.Struct:
		output := ""
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			output += fmt.Sprintf("/%s=%v", kind.Field(i).Name, f.Interface())
		}
		output += "$\n"
		return []byte(output)
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return []byte("Null")
		}
		return c.reflectValue(v.Elem())
	default:
		panic(v.Kind())
	}
	
	return []byte("???")
}

func (c *tcpClient) listener() {
	for {
		buffer := make([]byte, 1024)
		bytesRead, err := c.Conn.Read(buffer)
		if err != nil {
			fmt.Println("Read failed. Closing TCP client...")
			break
		}
		fmt.Println("[TCP] -> " + string(buffer[0:bytesRead]))
	}
	c.Conn.Close()
}

func (c *tcpClient) sender() {
	for message := range c.Send {
		// TODO: Remove unmarshalling :(
		/*str := ""
		err := json.Unmarshal(message, &str)
		if err != nil {
			fmt.Println("Unmarshalling failed.")
		}*/

		fmt.Println("[TCP] <- " + string(message))
		_, err := c.Conn.Write(message)
		if err != nil {
			fmt.Println("Send failed. Closing TCP client...")
			break
		}
	}
	c.Conn.Close()
}

/* ===|==================================
 * WebSocket Client 
 * ======================================*/
type webSocketClient struct {
	genericClient
	Conn *websocket.Conn
}

type InPacket struct {
	Event
	Sender *webSocketClient
}

type OutPacket struct {
	Name string `json:"name"`
	Message interface{} `json:"message"`
}

func (c *webSocketClient) Format(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err.Error())
	}
	return data
}

func (c *webSocketClient) listener() {
	for {
		var event string
		err := websocket.Message.Receive(c.Conn, &event)
		if err != nil {
			fmt.Println("Read failed. Closing WS client...")
			break
		}

		var e []Event
		
		if err := json.Unmarshal([]byte(event), &e); err != nil {
			fmt.Println("error:", err)
		}

		packet := InPacket{Event: e[0], Sender: c}
		events.incoming <- packet
	}
	c.Conn.Close()
}

func (c *webSocketClient) sender() {
	for message := range c.Send {

		msg := string(message)
		fmt.Println("[WS]  <- " + msg)
		
		if err := websocket.Message.Send(c.Conn, msg); err != nil {
			fmt.Println("Send failed. Closing socket...")
			break
		}
	}
	c.Conn.Close()
}