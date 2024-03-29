//go:build cgo
// +build cgo

package ed25519

/*
#include <stdbool.h>
bool intToBoolED25519(int i)
{
	if(i != 0){
	return true;
	} else {
	return false;
	}
}

#cgo CFLAGS: -DDEV_MODE=1 -DCONFIGURED=1
#cgo CFLAGS: -I${SRCDIR}/libsodium/src/libsodium/include/sodium
#cgo CFLAGS: -I${SRCDIR}/libsodium/src/libsodium/include/sodium/private
#include "./libsodium/src/libsodium/crypto_hash/sha512/cp/hash_sha512_cp.c"
#include "./libsodium/src/libsodium/crypto_sign/ed25519/ref10/open.c"
#include "./libsodium/src/libsodium/crypto_sign/crypto_sign.c"
#include "./libsodium/src/libsodium/crypto_sign/ed25519/sign_ed25519.c"
#include "./libsodium/src/libsodium/sodium/utils.c"
#include "./libsodium/src/libsodium/sodium/core.c"
#include "./libsodium/src/libsodium/sodium/runtime.c"
#include "./libsodium/src/libsodium/crypto_sign/ed25519/ref10/sign.c"
#include "./libsodium/src/libsodium/randombytes/randombytes.c"
#include "./libsodium/src/libsodium/crypto_core/ed25519/ref10/ed25519_ref10.c"
#include "./libsodium/src/libsodium/randombytes/sysrandom/randombytes_sysrandom.c"
#include "./libsodium/src/libsodium/crypto_pwhash/argon2/argon2-core.c"
#include "./libsodium/src/libsodium/crypto_pwhash/argon2/blake2b-long.c"
#include "./libsodium/src/libsodium/crypto_generichash/blake2b/ref/generichash_blake2b.c"
#include "./libsodium/src/libsodium/crypto_generichash/blake2b/ref/blake2b-ref.c"
#include "./libsodium/src/libsodium/crypto_generichash/blake2b/ref/blake2b-compress-ref.c"
#include "./libsodium/src/libsodium/crypto_pwhash/argon2/argon2-fill-block-ref.c"
#include "./libsodium/src/libsodium/crypto_verify/sodium/verify.c"
#include "./libsodium/src/libsodium/crypto_onetimeauth/poly1305/onetimeauth_poly1305.c"
#include "./libsodium/src/libsodium/crypto_onetimeauth/poly1305/donna/poly1305_donna.c"
#include "./libsodium/src/libsodium/crypto_scalarmult/curve25519/scalarmult_curve25519.c"
#include "./libsodium/src/libsodium/crypto_scalarmult/curve25519/ref10/x25519_ref10.c"
#include "./libsodium/src/libsodium/crypto_stream/chacha20/stream_chacha20.c"
#include "./libsodium/src/libsodium/crypto_stream/chacha20/ref/chacha20_ref.c"
#include "./libsodium/src/libsodium/crypto_stream/salsa20/stream_salsa20.c"
#include "./libsodium/src/libsodium/crypto_stream/salsa20/ref/salsa20_ref.c"
#include "./libsodium/src/libsodium/crypto_core/salsa/ref/core_salsa_ref.c"
#include "./libsodium/src/libsodium/crypto_sign/ed25519/ref10/keypair.c"
#include "./libsodium/src/libsodium/crypto_sign/ed25519/ref10/batchSupport.c"
#include "./libsodium/src/libsodium/include/sodium/private/ed25519_ref10.h"
#include "./libsodium/src/libsodium/crypto_sign/ed25519/ref10/open_bv_compat.c"
enum {
	sizeofPtr = sizeof(void*),
	sizeofULongLong = sizeof(unsigned long long),
};
*/
import "C"
import (
	"errors"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"unsafe"
)

func generateKeyPair() (sec []byte, pub []byte, err error) {
	sec = make([]byte, PrivateKeyBytes)
	pub = make([]byte, PublicKeyBytes)
	if C.intToBoolED25519(C.crypto_sign_keypair((*C.uchar)(unsafe.Pointer(&pub[0])), (*C.uchar)(unsafe.Pointer(&sec[0])))) {
		return nil, nil, errors.New("libsodium crypto_sign_keypair err")
	}
	return sec, pub, nil
}

func generateSeedWithLibsodium() []byte {
	random := make([]byte, KeyGenSeedBytes)
	C.randombytes_buf(unsafe.Pointer(&random[0]), KeyGenSeedBytes)
	return random
}

func generateKeyPairFromSeed(seed []byte) (sec []byte, pub []byte, err error) {
	if len(seed) != KeyGenSeedBytes {
		return nil, nil, errors.New("input invalid seed")
	}
	sec = make([]byte, PrivateKeyBytes)
	pub = make([]byte, PublicKeyBytes)
	if C.intToBoolED25519(C.crypto_sign_seed_keypair(
		(*C.uchar)(unsafe.Pointer(&pub[0])),
		(*C.uchar)(unsafe.Pointer(&sec[0])),
		(*C.uchar)(unsafe.Pointer(&seed[0])))) {
		return nil, nil, errors.New("libsodium crypto_sign_seed_keypair err")
	}
	return sec, pub, nil
}

func seckeyToPubkey(sec []byte) (pub []byte, err error) {
	if len(sec) != PrivateKeyBytes {
		return nil, errors.New("input invalid seckey")
	}
	pub = make([]byte, PublicKeyBytes)
	C.crypto_sign_ed25519_sk_to_pk((*C.uchar)(unsafe.Pointer(&pub[0])), (*C.uchar)(unsafe.Pointer(&sec[0])))
	return pub, nil
}

func signDetached(sec []byte, msg []byte) (sig []byte, err error) {
	if len(sec) != PrivateKeyBytes || len(msg) == 0 {
		return nil, errors.New("input invalid argument")
	}
	sig = make([]byte, SignatureBytes)
	var siglen C.ulonglong
	if C.intToBoolED25519(C.crypto_sign_detached(
		(*C.uchar)(unsafe.Pointer(&sig[0])),
		&siglen,
		(*C.uchar)(unsafe.Pointer(&msg[0])),
		(C.ulonglong)(uint64(len(msg))),
		(*C.uchar)(unsafe.Pointer(&sec[0])))) {
		return nil, errors.New("libsodium crypto_sign_detached err")
	}
	return sig, nil
}

func verifyDetached(pub []byte, msg []byte, sig []byte) (bool, error) {
	if len(pub) != PublicKeyBytes || len(msg) == 0 || len(sig) != SignatureBytes {
		return false, errors.New("input invalid argument")
	}
	if C.intToBoolED25519(C.crypto_sign_verify_detached(
		(*C.uchar)(unsafe.Pointer(&sig[0])),
		(*C.uchar)(unsafe.Pointer(&msg[0])),
		(C.ulonglong)(uint64(len(msg))),
		(*C.uchar)(unsafe.Pointer(&pub[0])))) {
		return false, errors.New("verify failed")
	}
	return true, nil
}

func batchVerify(publicKeys []tpcrtypes.PublicKey, messages [][]byte, signatures []tpcrtypes.Signature) bool {
	numberOfSignatures := len(messages)

	messagesAllocation := C.malloc(C.size_t(C.sizeofPtr * numberOfSignatures))
	messagesLenAllocation := C.malloc(C.size_t(C.sizeofULongLong * numberOfSignatures))
	publicKeysAllocation := C.malloc(C.size_t(C.sizeofPtr * numberOfSignatures))
	signaturesAllocation := C.malloc(C.size_t(C.sizeofPtr * numberOfSignatures))
	pass := C.malloc(C.size_t(C.sizeof_int * numberOfSignatures))

	defer func() {
		C.free(messagesAllocation)
		C.free(messagesLenAllocation)
		C.free(publicKeysAllocation)
		C.free(signaturesAllocation)
		C.free(pass)
	}()

	// load all the data pointers into the array pointers.
	for i := 0; i < numberOfSignatures; i++ {
		*(*uintptr)(unsafe.Pointer(uintptr(messagesAllocation) + uintptr(i*C.sizeofPtr))) = uintptr(unsafe.Pointer(&messages[i][0]))
		*(*C.ulonglong)(unsafe.Pointer(uintptr(messagesLenAllocation) + uintptr(i*C.sizeofULongLong))) = C.ulonglong(len(messages[i]))
		*(*uintptr)(unsafe.Pointer(uintptr(publicKeysAllocation) + uintptr(i*C.sizeofPtr))) = uintptr(unsafe.Pointer(&publicKeys[i][0]))
		*(*uintptr)(unsafe.Pointer(uintptr(signaturesAllocation) + uintptr(i*C.sizeofPtr))) = uintptr(unsafe.Pointer(&signatures[i][0]))
	}

	// call the batch verifier
	allPass := C.crypto_sign_ed25519_open_batch(
		(**C.uchar)(unsafe.Pointer(messagesAllocation)),
		(*C.ulonglong)(unsafe.Pointer(messagesLenAllocation)),
		(**C.uchar)(unsafe.Pointer(publicKeysAllocation)),
		(**C.uchar)(unsafe.Pointer(signaturesAllocation)),
		C.size_t(len(messages)),
		(*C.int)(unsafe.Pointer(pass)))

	return allPass == 0
}

func toCurve25519(sec []byte, pub []byte) (curveSec []byte, curvePub []byte, err error) {
	if len(sec) != PrivateKeyBytes || len(pub) != PublicKeyBytes {
		return nil, nil, errors.New("input invalid argument")
	}
	curveSec = make([]byte, Curve25519PrivateKeyBytes)
	curvePub = make([]byte, Curve25519PublicKeyBytes)
	C.crypto_sign_ed25519_sk_to_curve25519((*C.uchar)(&curveSec[0]), (*C.uchar)(&sec[0]))
	C.crypto_sign_ed25519_pk_to_curve25519((*C.uchar)(&curvePub[0]), (*C.uchar)(&pub[0]))
	return curveSec, curvePub, nil
}
