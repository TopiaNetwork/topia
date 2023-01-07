package rpc

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"time"

	tlog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
)

type methodType struct {
	isObjMethod        bool
	receiver           reflect.Value
	mValue             reflect.Value
	mType              reflect.Type
	errPos             int // err return idx, of -1 when method cannot return error, make sure error is the last return value of method
	authLevel          AuthLevel
	cacheAble          bool // determine whether the method can be cached
	cacheExpireSeconds int  //
	timeout            time.Duration
}

// Call
// errMsg: hold ErrMsg for method returned, caller should allocate space for it.
func (m *methodType) Call(in []reflect.Value, errMsg *ErrMsg) ([]byte, error) {
	if errMsg == nil {
		return nil, errors.New("nil errMsg Pointer")
	}

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

	var fullArgs []reflect.Value
	if m.isObjMethod {
		fullArgs = append(fullArgs, m.receiver)
		fullArgs = append(fullArgs, in...)
	} else {
		fullArgs = in
	}

	if m.mType.NumIn() != len(fullArgs) {
		mylog.Error("number of parameters error")
		mylog.Infof("hope: %v", m.mType.NumIn())
		mylog.Infof("actual: %v", len(fullArgs))
		return nil, ErrInput
	}
	for i := range fullArgs {
		if fullArgs[i].Type().Kind() == m.mType.In(i).Kind() {
			continue
		}
		if fullArgs[i].Type().ConvertibleTo(m.mType.In(i)) {
			fullArgs[i] = fullArgs[i].Convert(m.mType.In(i))
		} else {
			mylog.Error(ErrInput.Error() + " type of parameter error" + " input type:" + fullArgs[i].Type().Name() + " target type" + m.mType.In(i).Name())
			return nil, ErrInput
		}
	}
	back := m.mValue.Call(fullArgs)
	if len(back) == 0 { // if no return
		return nil, nil
	}

	// If func has err return part and return non-nil error
	if m.errPos >= 0 && !back[m.errPos].IsNil() {
		err := back[m.errPos].Interface().(error)
		*errMsg = ErrMsg{
			Errtype:   ErrServiceReturnError,
			ErrString: err.Error(),
		}
	}

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
