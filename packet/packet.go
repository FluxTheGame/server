package packet

import (
	"io"
	"fmt"
	"encoding/json"
	"bitbucket.org/jahfer/flux-middleman/tcp"
)

// What is received from a WebSocket
type In struct {
	Raw []byte
	Sender io.WriteCloser
}

// Sending out to a WebSocket
type Out struct {
	Name    string      `json:"name"`
	Message interface{} `json:"message"`
}

// In theory, this would unmarshal a response from both TCP and WS
// Right now, it only supports WS, due to JSON
func Unmarshal(b []byte, v interface{}) (err error) {

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[NOTICE]\tCaught malformed message to server\n")
		}
	}()

	if b[0] == '/' {
		//return unmarshalTCP(b, v)
		return tcp.UnmarshalAsEvent(b, v)
	}

	return json.Unmarshal(b, v)
}