package packet

import (
	"io"
	"runtime"
	"reflect"
	"errors"
	"strings"
	"strconv"
	"encoding/json"
	_ "fmt"
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
		//return unmarshalTCP(b, v)
		return unmarshalTCPAsEvent(b, v)
	}

	return json.Unmarshal(b, v)
}

// Not worth the dev time to make a full unmarshaller :(
func unmarshalTCPAsEvent(b []byte, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	if len(b) < 5 {
		return errors.New("Message too short. Malformed syntax.")
	}

	p := string(b)
	messages := strings.Split(p, "$")
	message := messages[0]

	datamap := parseToMap(message)
	
	item := reflect.ValueOf(v).Elem()
	kind := item.Kind()
	
	if kind == reflect.Slice {
		item.Set(reflect.MakeSlice(item.Type(), 1, 1))
	}

	evt := item.Index(0)

	//return matchMapToStruct(datamap, v)
	evt.FieldByName("Name").SetString(datamap["name"])
	evt.FieldByName("Args").SetBytes(b)

	item.Index(0).Set(evt)

	return nil
}

func parseToMap(raw string) (datamap map[string] string) {
	datamap = make(map[string] string)

	data := strings.Split(raw, "/")
	for _, m := range data {
		keyvalue := strings.Split(m, "=")
		if len(keyvalue) == 2 {
			datamap[keyvalue[0]] = keyvalue[1]
		}
	}

	return
}

func matchMapToStruct(datamap map[string] string, v interface{}) (err error) {

	rv := reflect.ValueOf(v)
	pv := rv
	if pv.Kind() != reflect.Ptr || pv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	st := pv.Elem()
	typeOfData := st.Type()

	for i := 0; i < st.NumField(); i++ {

		var fieldname string

		tag := typeOfData.Field(i).Tag.Get("tcp")
		if tag != "" {
			fieldname = tag
		} else {
			fieldname = typeOfData.Field(i).Name
		}

		if d, ok := datamap[fieldname]; ok {
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

func UnmarshalTCP(b []byte, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	if len(b) < 3 {
		return errors.New("Message too small. Malformed syntax.")
	}

	p := string(b)
	messages := strings.Split(p, "$")
	message := messages[0]

	datamap := parseToMap(message)
	return matchMapToStruct(datamap, v)
}

