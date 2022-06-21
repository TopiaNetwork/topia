package rpc

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	tlog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
)

type methodType struct {
	mValue    reflect.Value
	mType     reflect.Type
	authLevel byte
	cacheTime int //seconds
	timeout   time.Duration
}

func (m *methodType) Call(in []reflect.Value) ([]byte, error) {
	mylog, err := tlog.CreateMainLogger(logcomm.DebugLevel, tlog.JSONFormat, tlog.StdErrOutput, "")
	if err != nil {
		panic(err)
	}
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			buf = buf[:n]

			err := fmt.Errorf("[painc service internal error]: %v, method: %s, argv: %+v, stack: %s",
				r, m.mType.Name(), in, buf)
			mylog.Error(err.Error())
		}
	}()

	if m.mType.NumIn() != len(in) {
		mylog.Error("number of parameters error")
		return nil, ErrInput
	}
	for i := range in {
		if in[i].Type().Kind() == m.mType.In(i).Kind() {
			continue
		}
		if in[i].Type().ConvertibleTo(m.mType.In(i)) {
			in[i] = in[i].Convert(m.mType.In(i))
		} else {
			mylog.Error(ErrInput.Error() + " type of parameter error" + " input type:" + in[i].Type().Name() + " target type" + m.mType.In(i).Name())
			return nil, ErrInput
		}
	}
	back := m.mValue.Call(in)
	outArgs := make([]interface{}, len(back))
	for i := range back {
		outArgs[i] = back[i].Interface()
	}

	respBuf, err := Encode(outArgs)
	if err != nil {
		mylog.Error(err.Error())
		return nil, err
	}

	return respBuf, nil
}
