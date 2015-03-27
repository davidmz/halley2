package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"

	"github.com/davidmz/halley2/internal/channel"
	"github.com/davidmz/halley2/internal/npool"
	"github.com/davidmz/logg"
	"github.com/davidmz/memcache/simplemmc"
)

type HandlerMemc struct {
	Conf *Conf           `inject:""`
	Log  *logg.Logger    `inject:""`
	Pool npool.NamedPool `inject:""`
}

func (h *HandlerMemc) Get(key string) ([]byte, error) {
	h.Log.TRACE("mmc get %q", key)

	if key != "token" {
		return nil, simplemmc.ErrNotFound
	}

	b, _ := json.Marshal(&struct {
		Token   []byte `json:"token"`
		Expires int64  `json:"expires"`
	}{
		Expires: TOKEN_EXP_TIME,
		Token:   NewToken(h.Conf.Secret),
	})

	return b, nil
}

func (h *HandlerMemc) Set(key string, data []byte, mode simplemmc.SetMode) error {

	h.Log.TRACE("mmc set %q", key)

	var site *SiteConf
	for _, st := range h.Conf.Sites {
		if st.Name == key {
			site = st
			break
		}
	}

	if site == nil {
		return errors.New("site not found")
	}

	pReq := new(PostRequest)
	if err := json.Unmarshal(data, pReq); err != nil {
		return err
	}

	if !CheckToken(pReq.Token, h.Conf.Secret) {
		return errors.New("Invalid token")
	}

	mac := hmac.New(sha256.New, site.PostSecret)
	mac.Write(pReq.Token)
	sign := mac.Sum(nil)

	if !hmac.Equal(sign, pReq.Signature) {
		return errors.New("Invalid signature")
	}

	ch := h.Pool.Get(channel.ChanKey{
		Site: site.Name,
		Name: pReq.ChanName,
	}).(*channel.Channel)

	ch.AddMessage(pReq.MessageBody)

	return nil
}

func (h *HandlerMemc) Del(key string) error { return nil }
