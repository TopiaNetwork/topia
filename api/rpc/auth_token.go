package rpc

import "time"

const (
	None    byte = 0x00
	Read    byte = 0x01
	Write   byte = 0x02
	Sign    byte = 0x04
	Manager byte = 0x08
)

type AuthObject struct {
	// authority => token
	tokenArr map[byte]string
}

func NewAuthObject(m map[byte]string) *AuthObject {
	return &AuthObject{
		tokenArr: m,
	}
}

func (auth *AuthObject) Token(authority byte) (token string, err error) {
	token, ok := auth.tokenArr[authority]
	if !ok {
		token = RandStringRunes(10) + time.Stamp // TODO 这里的 time.Stamp想表达的意思不对吧？
		auth.tokenArr[authority] = token
	}
	return token, err
}

func (auth *AuthObject) Level(token string) (authority byte) {
	for k, v := range auth.tokenArr {
		if token == v {
			return k
		}
	}
	return None
}
