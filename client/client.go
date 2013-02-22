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

type Client interface {
	// Send data to client
	io.WriteCloser
	// Translate from struct to specified format
	Format(v interface{}) []byte
}

/* ===|====================================================
 * Generic Client  
 * -> Used to provide generic handlers that fit Client{}
 * ======================================================*/
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

// TCP client doesn't read JSON! :O
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

			key := kind.Field(i).Name

			tv := kind.Field(i).Tag.Get("tcp")
			if tv != "" {
				if tv == "-" {
					continue
				}

				key = tv
			}

			output += fmt.Sprintf("/%s=%v", key, f.Interface())
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
		bytesRead, err := c.Conn.Read(buffer)
		if err != nil {
			break
		}
		//fmt.Println("[TCP] -> " + string(buffer[0:bytesRead]))

		pkt := packet.In{Raw: buffer[0:bytesRead], Sender: c}

		/*c := collector{}
		packet.Unmarshal(pkt.Raw, &c)
		fmt.Println(c)*/

		incoming <- pkt
	}
	c.Conn.Close()
}

// Oh, you want to send something out? Fiiiiine...
func (c *TcpClient) Sender() {
	for message := range c.Send {
		_, err := c.Conn.Write(message)
		if err != nil {
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

		if err := websocket.Message.Send(c.Conn, msg); err != nil {
			break
		}
	}
	c.Conn.Close()
}
