package packet

import (
	"io"
	"runtime"
	"reflect"
	"errors"
	"strings"
	"strconv"
	"encoding/json"
	"fmt"
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

type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "tcp: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "tcp: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "tcp: Unmarshal(nil " + e.Type.String() + ")"
}

// In theory, this would unmarshal a response from both TCP and WS
// Right now, it only supports WS, due to JSON
func Unmarshal(b []byte, v interface{}) (err error) {
	if b[0] == '/' {
		return unmarshalTCP(b, v)
	}

	return json.Unmarshal(b, v)
}

func unmarshalTCP(b []byte, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	rv := reflect.ValueOf(v)
	pv := rv
	if pv.Kind() != reflect.Ptr || pv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	if len(b) < 3 {
		return errors.New("Message too small. Malformed syntax.")
	}

	st := pv.Elem()
	typeOfData := st.Type()

	p := string(b)
	messages := strings.Split(p, "$")
	message := messages[0]

	tmpData := make(map[string] string)

	data := strings.Split(message, "/")
	for _, m := range data {
		keyvalue := strings.Split(m, "=")
		if len(keyvalue) == 2 {
			tmpData[keyvalue[0]] = keyvalue[1]
		}
	}

	for i := 0; i < st.NumField(); i++ {
		if d, ok := tmpData[typeOfData.Field(i).Name]; ok {
			switch st.Field(i).Kind() {
			case reflect.String:
				st.Field(i).SetString(d)
			case reflect.Int:
				digit, _ := strconv.Atoi(d)
				st.Field(i).SetInt(int64(digit))
			}
		} else {
			return errors.New("Did not fulfill all members of structure.")
		}
	}

	return nil

}

