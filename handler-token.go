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
	EXP_TIME = 120
)

func (h *HandlerToken) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	exp := time.Now().Unix() + EXP_TIME

	mac := hmac.New(sha256.New, h.Conf.Secret)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, exp)
	binary.Write(mac, binary.LittleEndian, exp)
	buf.Write(mac.Sum(nil))

	sendOK(w, &struct {
		Token   []byte `json:"token"`
		Expires int64  `json:"expires"`
	}{
		Expires: EXP_TIME,
		Token:   buf.Bytes(),
	})
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
