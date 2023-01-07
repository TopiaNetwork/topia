package rpc

import "time"

type AuthLevel byte

const (
	None    AuthLevel = 0
	Read    AuthLevel = 1
	Write   AuthLevel = 2
	Sign    AuthLevel = 3
	Manager AuthLevel = 4
)

type AuthObject struct {
	// authority => token
	tokenArr map[AuthLevel]string
}

func NewAuthObject(m map[AuthLevel]string) *AuthObject {
	return &AuthObject{
		tokenArr: m,
	}
}

func (auth *AuthObject) Token(authority AuthLevel) (token string, err error) {
	token, ok := auth.tokenArr[authority]
	if !ok {
		token = RandStringRunes(10) + time.Stamp // TODO 这里的 time.Stamp想表达的意思不对吧？
		auth.tokenArr[authority] = token
	}
	return token, err
}

func (auth *AuthObject) Level(token string) (authority AuthLevel) {
	for k, v := range auth.tokenArr {
		if token == v {
			return k
		}
	}
	return None
}
