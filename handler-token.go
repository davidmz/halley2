package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"net/http"
	"time"

	"github.com/davidmz/logg"
)

type HandlerToken struct {
	Conf *Conf        `inject:""`
	Log  *logg.Logger `inject:""`
}

const (
	TOKEN_EXP_TIME = 120
)

func (h *HandlerToken) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sendOK(w, &struct {
		Token   []byte `json:"token"`
		Expires int64  `json:"expires"`
	}{
		Expires: TOKEN_EXP_TIME,
		Token:   NewToken(h.Conf.Secret),
	})
}

func NewToken(secret []byte) []byte {
	exp := time.Now().Unix() + TOKEN_EXP_TIME

	mac := hmac.New(sha256.New, secret)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, exp)
	binary.Write(mac, binary.LittleEndian, exp)
	buf.Write(mac.Sum(nil))

	return buf.Bytes()
}

func CheckToken(token, secret []byte) bool {
	if len(token) != 8+sha256.Size {
		return false
	}
	ts := binary.LittleEndian.Uint64(token[:8])
	mac := hmac.New(sha256.New, secret)
	mac.Write(token[:8])
	return hmac.Equal(mac.Sum(nil), token[8:]) && uint64(time.Now().Unix()) < ts
}
