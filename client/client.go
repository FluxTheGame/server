package client

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"bitbucket.org/jahfer/flux-middleman/packet"
)

/* ===|==================================
 * Client Interface
 * ======================================*/
type Client interface {
	// Send data to client
	io.WriteCloser
	// Translate from struct to specified format
	Format(v interface{}) []byte
}

/* ===|==================================
 * Generic Client 
 * ======================================*/
type genericClient struct {
	Conn io.WriteCloser
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

// By default, we just assume everything is going to be JSON
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
type TcpClient struct {
	genericClient
	Conn net.Conn
}

// TCP client isn't JSON! :O
func (c *TcpClient) Format(v interface{}) []byte {
	ref := reflect.ValueOf(&v)
	return c.reflectValue(ref)
}

// Lets take a look inside the object, and format it how we'd like
func (c *TcpClient) reflectValue(v reflect.Value) []byte {

	kind := v.Type()

	switch v.Kind() {
	case reflect.Struct:
		output := ""
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			output += fmt.Sprintf("/%s=%v", kind.Field(i).Name, f.Interface())
		}
		output += "$"//\n
		return []byte(output)
	// Lets try it again, using the elements inside
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return []byte("Null")
		}
		return c.reflectValue(v.Elem())
	// Don't send me a non-struct!
	default:
		panic(v.Kind())
	}

	return []byte("???")
}

// Always have to keep an eye out for creepers...
func (c *TcpClient) Listener(incoming chan packet.In) {
	for {
		buffer := make([]byte, 1024)
		_, err := c.Conn.Read(buffer)
		if err != nil {
			//fmt.Println("Read failed. Closing TCP client...")
			break
		}
		//fmt.Println("[TCP] -> " + string(buffer[0:bytesRead]))

		//packet := packet.In{Raw: buffer[0:bytesRead], Sender: c}
		//incoming <- packet
	}
	c.Conn.Close()
}

// Oh, you want to send something out? Fiiiiine...
func (c *TcpClient) Sender() {
	for message := range c.Send {
		_, err := c.Conn.Write(message)
		if err != nil {
			//fmt.Println("Send failed. Closing TCP client...")
			break
		}
	}
	c.Conn.Close()
}

/* ===|==================================
 * WebSocket Client 
 * ======================================*/
type WebSocketClient struct {
	genericClient
	Conn *websocket.Conn
}

// Catch any traffic directed this way
func (c *WebSocketClient) Listener(incoming chan packet.In) {

	defer func(){ c.Conn.Close() }()

	for {
		var event string
		err := websocket.Message.Receive(c.Conn, &event)
		if err != nil {
			//fmt.Println("Read failed. Closing WS client...")
			break
		}

		packet := packet.In{Raw: []byte(event), Sender: c}
		incoming <- packet
	}

	deadPacket := packet.In{Raw: nil, Sender: c}
	incoming <- deadPacket
}

// Dispatch data sent into the connection
func (c *WebSocketClient) Sender() {
	for message := range c.Send {

		msg := string(message)
		//fmt.Println("[WS]  <- " + msg)

		if err := websocket.Message.Send(c.Conn, msg); err != nil {
			//fmt.Println("Send failed. Closing socket...")
			break
		}
	}
	c.Conn.Close()
}
