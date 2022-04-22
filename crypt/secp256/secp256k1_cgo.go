package secp256

/*
#cgo CFLAGS: -DECMULT_WINDOW_SIZE=15 -DECMULT_GEN_PREC_BITS=4
#cgo LDFLAGS: -lstdc++

typedef void (*secp256k1CallBack)(const char *, void*);
extern void secp256k1IllegalCallBack(char*, void*);
extern void secp256k1ErrorCallBack(char*, void*);
#include <stdbool.h>
bool intToBool(int i)
{
	if(i != 0){
	return true;
	} else {
	return false;
	}
}
#include "./secp256k1/src/secp256k1.c"
#include "./secp256k1/src/precomputed_ecmult.c"
#include "./secp256k1/src/precomputed_ecmult_gen.c"
#include "./secp256k1/src/modules/recovery/main_impl.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

type secp256k1ContextPointer struct {
	p *C.secp256k1_context
}
type secp256k1EcdsaRecoverableSignature struct {
	v C.secp256k1_ecdsa_recoverable_signature
}
type secp256k1Pubkey struct {
	v C.secp256k1_pubkey
}
type secp256k1EcdsaSignature struct {
	v C.secp256k1_ecdsa_signature
}

const secp256k1PubkeyOriginalSize = 64 //64 bytes

func contextCreate() secp256k1ContextPointer {
	var ret secp256k1ContextPointer
	ret.p = C.secp256k1_context_create(C.SECP256K1_CONTEXT_SIGN | C.SECP256K1_CONTEXT_VERIFY)
	C.secp256k1_context_set_illegal_callback(ret.p, (C.secp256k1CallBack)(unsafe.Pointer(C.secp256k1IllegalCallBack)), nil)
	C.secp256k1_context_set_error_callback(ret.p, (C.secp256k1CallBack)(unsafe.Pointer(C.secp256k1ErrorCallBack)), nil)
	return ret
}

func contextDestroy(ctx secp256k1ContextPointer) {
	C.secp256k1_context_destroy(ctx.p)
}

func contextRandomize(ctx secp256k1ContextPointer, random []byte) error {
	if len(random) != 32 {
		return errors.New("input invalid parameter")
	}
	if !(C._Bool)(C.intToBool(C.secp256k1_context_randomize(ctx.p, (*C.uchar)(unsafe.Pointer(&random[0]))))) {
		return errors.New("failed to context randomize")
	}
	return nil
}

func ecSeckeyVerify(ctx secp256k1ContextPointer, seckey []byte) (bool, error) {
	if len(seckey) != PrivateKeyBytes {
		return false, errors.New("input invalid parameter")
	}
	if !(C._Bool)(C.intToBool(C.secp256k1_ec_seckey_verify(ctx.p, (*C.uchar)(unsafe.Pointer(&seckey[0]))))) {
		return false, nil
	}
	return true, nil
}

func ecPubkeyCreateAndSerialize(ctx secp256k1ContextPointer, seckey []byte) (pubkey [PublicKeyBytes]byte, err error) {
	var pub secp256k1Pubkey

	if len(seckey) != PrivateKeyBytes {
		return [PublicKeyBytes]byte{}, errors.New("input invalid parameter")
	}

	if !(C._Bool)(C.intToBool(C.secp256k1_ec_pubkey_create(ctx.p, &pub.v, (*C.uchar)(unsafe.Pointer(&seckey[0]))))) { // != 1
		return [65]byte{}, errors.New("failed to create pubkey")
	}

	pubkeyLen := uint(PublicKeyBytes)
	if !(C._Bool)(C.intToBool(C.secp256k1_ec_pubkey_serialize(ctx.p, (*C.uchar)(unsafe.Pointer(&pubkey[0])), (*C.size_t)(unsafe.Pointer(&pubkeyLen)), &pub.v, C.SECP256K1_EC_UNCOMPRESSED))) {
		return [65]byte{}, errors.New("failed to serialize pubkey")
	}
	return pubkey, nil
}

func ecdsaSignAndSerialize(ctx secp256k1ContextPointer, seckey []byte, msg []byte) (serializedSig [SignatureRecoverableBytes]byte, err error) {
	var recSig secp256k1EcdsaRecoverableSignature
	var recID int32

	if len(seckey) != PrivateKeyBytes || len(msg) != MsgBytes {
		return [SignatureRecoverableBytes]byte{}, errors.New("input invalid parameter")
	}

	if !(C._Bool)(C.intToBool(C.secp256k1_ecdsa_sign_recoverable(ctx.p, &recSig.v, (*C.uchar)(unsafe.Pointer(&msg[0])), (*C.uchar)(unsafe.Pointer(&seckey[0])), C.secp256k1_nonce_function_rfc6979, nil))) {
		return [SignatureRecoverableBytes]byte{}, errors.New("failed to sign")
	}
	if !(C._Bool)(C.intToBool(C.secp256k1_ecdsa_recoverable_signature_serialize_compact(ctx.p, (*C.uchar)(unsafe.Pointer(&serializedSig[0])), (*C.int)(&recID), &recSig.v))) {
		return [SignatureRecoverableBytes]byte{}, errors.New("failed to serialize signature")
	}
	serializedSig[SignatureRecoverableBytes-1] = byte(recID)
	return serializedSig, nil
}

func ecdsaVerify(ctx secp256k1ContextPointer, pubkey []byte, signData []byte, msg []byte) (bool, error) {
	var sig secp256k1EcdsaSignature
	var pub secp256k1Pubkey

	if len(pubkey) != PublicKeyBytes || len(msg) != MsgBytes || len(signData) > SignatureRecoverableBytes || len(signData) < SignatureRecoverableBytes-1 {
		return false, errors.New("input invalid parameter")
	}

	if !(C._Bool)(C.intToBool(C.secp256k1_ecdsa_signature_parse_compact(ctx.p, &sig.v, (*C.uchar)(unsafe.Pointer(&signData[0]))))) {
		return false, errors.New("failed to parse signature")
	}
	if !(C._Bool)(C.intToBool(C.secp256k1_ec_pubkey_parse(ctx.p, &pub.v, (*C.uchar)(unsafe.Pointer(&pubkey[0])), C.size_t(uint(len(pubkey)))))) {
		return false, errors.New("secp256 Verify failed to parse pubkey")
	}
	if !(C._Bool)(C.intToBool(C.secp256k1_ecdsa_verify(ctx.p, &sig.v, (*C.uchar)(unsafe.Pointer(&msg[0])), &pub.v))) {
		return false, errors.New("secp256 Verify failed to verify")
	}
	return true, nil
}

func ecdsaRecoverPubkey(ctx secp256k1ContextPointer, msg []byte, signData []byte) (pubkey [PublicKeyBytes]byte, err error) {
	if len(msg) != MsgBytes || len(signData) > SignatureRecoverableBytes || len(signData) < SignatureRecoverableBytes-1 {
		return [PublicKeyBytes]byte{}, errors.New("input invalid parameter")
	}

	recSigPointer := C.malloc(C.size_t(SignatureRecoverableBytes))
	pubkeyPointer := C.malloc(C.size_t(secp256k1PubkeyOriginalSize))
	defer C.free(recSigPointer)
	defer C.free(pubkeyPointer)

	if !(C._Bool)(C.intToBool(C.secp256k1_ecdsa_recoverable_signature_parse_compact(ctx.p, (*C.secp256k1_ecdsa_recoverable_signature)(recSigPointer), (*C.uchar)(unsafe.Pointer(&signData[0])), C.int(int32(signData[SignatureRecoverableBytes-1]))))) {
		return [PublicKeyBytes]byte{}, errors.New("signature could not be parsed to recoverable signature")
	}
	if !(C._Bool)(C.intToBool(C.secp256k1_ecdsa_recover(ctx.p, (*C.secp256k1_pubkey)(pubkeyPointer), (*C.secp256k1_ecdsa_recoverable_signature)(recSigPointer), (*C.uchar)(unsafe.Pointer(&msg[0]))))) {
		return [PublicKeyBytes]byte{}, errors.New("public key failed to recover")
	}
	pubkeyLen := uint(PublicKeyBytes)
	if !(C._Bool)(C.intToBool(C.secp256k1_ec_pubkey_serialize(ctx.p, (*C.uchar)(unsafe.Pointer(&pubkey[0])), (*C.size_t)(unsafe.Pointer(&pubkeyLen)), (*C.secp256k1_pubkey)(pubkeyPointer), C.SECP256K1_EC_UNCOMPRESSED))) {
		return [PublicKeyBytes]byte{}, errors.New("failed to serialize pubkey")
	}
	return pubkey, nil
}
