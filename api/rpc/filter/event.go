package filter

import (
	"github.com/TopiaNetwork/topia/eventhub"
	"reflect"
	"strings"
)

type eventDataRef struct {
	t reflect.Type
	v reflect.Value
}

var eventMsgRegistry = make(map[string]eventDataRef)

// msg.Data should be struct or struct pointer
func registerEventMsgs(msgs ...eventhub.EventMsg) {
	for _, msg := range msgs {
		msgDataT := reflect.TypeOf(msg.Data)
		msgDataV := reflect.ValueOf(msg.Data)
		if msgDataT.Kind() == reflect.Ptr {
			msgDataV = msgDataV.Elem()
		}

		eventMsgRegistry[msg.Name] = eventDataRef{
			t: msgDataV.Type(),
			v: msgDataV,
		}
	}
}

func isValidFieldName(eventName string, fieldName string) (reflect.Kind, bool) {
	fieldName = strings.ToLower(fieldName)

	dataRef, ok := eventMsgRegistry[eventName]
	if !ok {
		return reflect.Invalid, false
	}

	return findFieldName(dataRef.t, dataRef.v, fieldName)
}

// make sure t,v is struct's type and value
func findFieldName(t reflect.Type, v reflect.Value, fieldName string) (reflect.Kind, bool) {
	for i := 0; i < t.NumField(); i++ {
		if v.Field(i).Kind() == reflect.Struct {
			innerV := v.Field(i)
			return findFieldName(innerV.Type(), innerV, fieldName)
		} else if v.Field(i).Kind() == reflect.Ptr && v.Field(i).Elem().Kind() == reflect.Struct {
			innerV := v.Field(i).Elem()
			return findFieldName(innerV.Type(), innerV, fieldName)
		} else {
			if strings.ToLower(t.Field(i).Name) == fieldName {
				if v.Field(i).Kind() == reflect.Slice { // only []byte can be found when slice
					if v.Field(i).Type().Elem().Kind() != reflect.Uint8 {
						continue
					}
				}

				return v.Field(i).Kind(), true
			}
		}
	}

	return reflect.Invalid, false

}
