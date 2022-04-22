package secp256

/*
void secp256k1IllegalCallBack(char*, void*);
void secp256k1ErrorCallBack(char*, void*);
*/
import "C"
import (
	"unsafe"
)

//export secp256k1IllegalCallBack
func secp256k1IllegalCallBack(message *C.char, data unsafe.Pointer) {
	illegalCallBackMsg := C.GoString(message)
	panic("secp256k1 illegal:" + illegalCallBackMsg)
}

//export secp256k1ErrorCallBack
func secp256k1ErrorCallBack(message *C.char, data unsafe.Pointer) {
	errorCallBackMsg := C.GoString(message)
	panic("secp256k1 error:" + errorCallBackMsg)
}
