package packet

import (
	"io"
	"encoding/json"
)

type In struct {
	Raw []byte
	Sender io.WriteCloser
}

type Out struct {
	Name    string      `json:"name"`
	Message interface{} `json:"message"`
}

func Unmarshal(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}