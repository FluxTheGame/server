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
type TcpClient struct {
	genericClient
	Conn net.Conn
}

func (c *TcpClient) Format(v interface{}) []byte {
	ref := reflect.ValueOf(&v)
	return c.reflectValue(ref)
}

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

func (c *TcpClient) Listener(incoming chan packet.In) {
	for {
		buffer := make([]byte, 1024)
		_, err := c.Conn.Read(buffer)
		if err != nil {
			fmt.Println("Read failed. Closing TCP client...")
			break
		}
		//fmt.Println("[TCP] -> " + string(buffer[0:bytesRead]))

		//packet := packet.In{Raw: buffer[0:bytesRead], Sender: c}
		//incoming <- packet
	}
	c.Conn.Close()
}

func (c *TcpClient) Sender() {
	for message := range c.Send {
		// TODO: add event emmiting

		//fmt.Printf("[TCP]\t<- " + string(message) + "\n")
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
type WebSocketClient struct {
	genericClient
	Conn *websocket.Conn
}

func (c *WebSocketClient) Format(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err.Error())
	}
	return data
}

func (c *WebSocketClient) Listener(incoming chan packet.In) {
	for {
		var event string
		err := websocket.Message.Receive(c.Conn, &event)
		if err != nil {
			fmt.Println("Read failed. Closing WS client...")
			break
		}

		packet := packet.In{Raw: []byte(event), Sender: c}
		incoming <- packet
	}
	c.Conn.Close()
}

func (c *WebSocketClient) Sender() {
	for message := range c.Send {

		msg := string(message)
		//fmt.Println("[WS]  <- " + msg)

		if err := websocket.Message.Send(c.Conn, msg); err != nil {
			fmt.Println("Send failed. Closing socket...")
			break
		}
	}
	c.Conn.Close()
}
