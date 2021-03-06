package tcp

import (
	"runtime"
	"reflect"
	"errors"
	"strings"
	"strconv"
)

func Unmarshal(b []byte, v interface{}) (err error) {
	defer catchError()

	if err := checkValid(b); err != nil {
		return err
	}

	message := splitPackets(b)
	datamap := parseToMap(message)
	return matchMapToStruct(datamap, v)
}

// can't parse multi-level messages, so manually do it
func UnmarshalAsEvent(b []byte, v interface{}) (err error) {
	defer catchError()

	if err := checkValid(b); err != nil {
		return err
	}

	message := splitPackets(b)
	datamap := parseToMap(message)
	
	item := reflect.ValueOf(v).Elem()
	kind := item.Kind()
	
	if kind == reflect.Slice {
		item.Set(reflect.MakeSlice(item.Type(), 1, 1))
	}

	evt := item.Index(0)

	evt.FieldByName("Name").SetString(datamap["name"])
	evt.FieldByName("Args").SetBytes(b)

	item.Index(0).Set(evt)

	return nil
}

func splitPackets(raw []byte) (s string) {
	p := string(raw)
	messages := strings.Split(p, "$")
	s = messages[0]

	return
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
			default:
				return errors.New("Unsupported type in interface{} (not int or string)")
			case reflect.String:
				st.Field(i).SetString(d)
			case reflect.Int:
				digit, _ := strconv.Atoi(d)
				st.Field(i).SetInt(int64(digit))
			}
		} else {
			return errors.New("Did not fulfill all members of structure")
		}
	}

	return nil
}

func catchError() (err error) {
	if r := recover(); r != nil {
		if _, ok := r.(runtime.Error); ok {
			panic(r)
		}
		err = r.(error)
	}

	return
}

func checkValid(data []byte) (err error) {
	if len(data) < 5 {
		return errors.New("Message too small. Malformed syntax.")
	}

	return nil
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