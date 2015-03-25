package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"io"
	"sync/atomic"

	"github.com/davidmz/halley2/internal/channel"
	"github.com/davidmz/halley2/internal/npool"
	"github.com/davidmz/logg"
)

type CmdRequest struct {
	Request   string          `json:"request"`
	RequestId string          `json:"req_id"`
	Body      json.RawMessage `json:"body"`
}

func (c *CmdRequest) RespOK() *CmdResponse {
	return &CmdResponse{Status: RespStatusOK, RequestId: c.RequestId}
}
func (c *CmdRequest) RespErr(msg interface{}) *CmdResponse {
	return &CmdResponse{Status: RespStatusErr, RequestId: c.RequestId, Body: msg}
}

type CmdSubscribe struct {
	Channel string      `json:"channel"`
	After   channel.Ord `json:"after"`
	Token   []byte      `json:"token"`
	Auth    []byte      `json:"auth"`
}

type CmdUnsubscribe struct {
	Channel string `json:"channel"`
}

const (
	RespStatusOK  = "ok"
	RespStatusErr = "error"
)

type CmdResponse struct {
	Status    string      `json:"response"`
	RequestId string      `json:"req_id,omitempty"`
	Body      interface{} `json:"body"`
}

type JSONio interface {
	ReadJSON(v interface{}) error
	WriteJSON(v interface{}) error
}

type Map map[string]interface{}

type Session struct {
	JSONio
	Log  *logg.Logger    `inject:""`
	Site *SiteConf       `inject:""`
	Conf *Conf           `inject:""`
	Pool npool.NamedPool `inject:""`

	rChan chan json.RawMessage
	wChan chan interface{}
	qChan chan struct{}
}

func NewSession(jio JSONio) *Session {
	return &Session{
		JSONio: jio,
		rChan:  make(chan json.RawMessage),
		wChan:  make(chan interface{}),
		qChan:  make(chan struct{}),
	}
}

func (s *Session) Run() {
	s.Log.TRACE("Session started")
	defer s.Log.TRACE("Session closed")

	atomic.AddUint32(&s.Conf.NSessions, 1)
	defer atomic.AddUint32(&s.Conf.NSessions, ^uint32(0))

	go s.reader()
	go s.writer()

	subscrNames := make(map[string]struct{})

L:
	for {
		req := new(CmdRequest)
		select {
		case m := <-s.rChan:
			json.Unmarshal(m, req)
			s.Log.TRACE("Message %q => %v", req.Request, string(m))

			switch req.Request {
			case "subscribe":
				cmd := new(CmdSubscribe)
				json.Unmarshal(req.Body, cmd)

				mac := hmac.New(sha256.New, s.Site.Secret)
				mac.Write([]byte(cmd.Channel))
				mac.Write(cmd.Token)
				sign := mac.Sum(nil)

				if !CheckToken(cmd.Token, s.Conf.Secret) {
					s.wChan <- req.RespErr("invalid token")
				} else if !hmac.Equal(sign, cmd.Auth) {
					s.wChan <- req.RespErr("invalid signature")
				} else if _, ok := subscrNames[cmd.Channel]; ok {
					s.wChan <- req.RespErr("already subscribed")
				} else {
					s.wChan <- req.RespOK()

					ch := s.Pool.Get(channel.ChanKey{
						Site: s.Site.Name,
						Name: cmd.Channel,
					}).(*channel.Channel)

					ch.Subscribe(s, cmd.After)
					subscrNames[cmd.Channel] = struct{}{}
				}

			case "unsubscribe":
				cmd := new(CmdUnsubscribe)
				json.Unmarshal(m, cmd)

				// не подписаны ли мы уже?
				if _, ok := subscrNames[cmd.Channel]; !ok {
					s.wChan <- req.RespErr("not subscribed")
				} else {
					ch := s.Pool.Get(channel.ChanKey{
						Site: s.Site.Name,
						Name: cmd.Channel,
					}).(*channel.Channel)

					ch.Unsubscribe(s)
					delete(subscrNames, cmd.Channel)

					s.wChan <- req.RespOK()
				}
			}

		case <-s.qChan:
			break L
		}
	}

	// Отписываемся
	for name := range subscrNames {
		ch := s.Pool.Get(channel.ChanKey{
			Site: s.Site.Name,
			Name: name,
		}).(*channel.Channel)
		ch.Unsubscribe(s)
	}

}

func (s *Session) ReceiveMessage(m *channel.Message) {
	s.wChan <- m
}

func (s *Session) reader() {
	for {
		var m json.RawMessage
		if err := s.ReadJSON(&m); err != nil {
			if err != io.EOF {
				s.Log.DEBUG("WS read error %v", err)
			}
			close(s.qChan)
		}
		select {
		case s.rChan <- m:
		case <-s.qChan:
			return
		}
	}
}

func (s *Session) writer() {
	for {
		select {
		case m := <-s.wChan:
			if err := s.WriteJSON(m); err != nil {
				s.Log.DEBUG("WS write error %v", err)
				close(s.qChan)
			}
		case <-s.qChan:
			return
		}
	}
}
