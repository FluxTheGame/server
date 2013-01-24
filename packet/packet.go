package packet

import (
	"io"
	"encoding/json"
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
func Unmarshal(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}