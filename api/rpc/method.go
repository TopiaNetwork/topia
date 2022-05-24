package rpc

import (
	"fmt"
	"log"
	"reflect"
	"runtime"
)

type methodType struct {
	mValue    reflect.Value
	mType     reflect.Type
	authLevel int
	cacheTime int
}

func (m *methodType) Call(in []reflect.Value) ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			buf = buf[:n]

			err := fmt.Errorf("[painc service internal error]: %v, method: %s, argv: %+v, stack: %s",
				r, m.mType.Name(), in, buf)
			log.Println(err)
		}
	}()

	if m.mType.NumIn() != len(in) {
		log.Print("number of parameters error")
		return nil, ErrInput
	}
	for i := range in {
		if in[i].Type().Kind() == m.mType.In(i).Kind() {
			continue
		}
		if in[i].Type().ConvertibleTo(m.mType.In(i)) {
			in[i] = in[i].Convert(m.mType.In(i))
		} else {
			log.Print(ErrInput.Error() + " type of parameter error" + " input type:" + in[i].Type().Name() + " target type" + m.mType.In(i).Name())
			return nil, ErrInput
		}
	}
	back := m.mValue.Call(in)
	log.Print(back)
	outArgs := make([]interface{}, len(back))
	for i := range back {
		outArgs[i] = back[i].Interface()
	}

	respBuf, err := Encode(outArgs)
	if err != nil {
		log.Print(err.Error())
		return nil, err
	}

	return respBuf, nil
}
