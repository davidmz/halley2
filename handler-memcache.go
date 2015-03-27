package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/davidmz/halley2/internal/channel"
	"github.com/davidmz/halley2/internal/npool"
	"github.com/davidmz/logg"
	"github.com/davidmz/memcache"
)

type HandlerMemc struct {
	Conf *Conf           `inject:""`
	Log  *logg.Logger    `inject:""`
	Pool npool.NamedPool `inject:""`
}

const MemcacheVersion = "1.0.0"

func (h *HandlerMemc) ServeMemcache(req *memcache.Request, resp *memcache.Response) error {
	log := h.Log.ChildWithPrefix("Memcache")
	log.TRACE("Command %q", req.Command)

	switch req.Command {

	case "set", "add", "replace", "append", "prepend":
		if err := h.handlePost(req); err != nil {
			log.DEBUG("Set error: %v", err)
			resp.ClientError(err.Error())
		} else {
			resp.Status("STORED")
		}

	case "get":
		if len(req.Args) == 0 {
			resp.ClientError("key required")
			break
		}

		log.TRACE("get %q", req.Args[0])

		switch req.Args[0] {
		case "token":
			b, _ := json.Marshal(&struct {
				Token   []byte `json:"token"`
				Expires int64  `json:"expires"`
			}{
				Expires: TOKEN_EXP_TIME,
				Token:   NewToken(h.Conf.Secret),
			})

			resp.Value(req.Args[0], b)
			resp.Status("END")

		default:
			resp.NotFound()

		}

	case "version":
		resp.Status("VERSION " + MemcacheVersion)

	case "quit":
		return memcache.ErrCloseConnection

	default:
		log.DEBUG("Unknown command %q", req.Command)
		resp.UnknownCommandError()

	}

	return nil
}

func (h *HandlerMemc) handlePost(req *memcache.Request) error {
	if len(req.Args) < 4 {
		return errors.New("invalid argument count")
	}

	dataSize, err := strconv.Atoi(req.Args[3])
	if dataSize <= 0 || err != nil {
		return errors.New("invalid data size")
	}

	data, err := req.ReadBody(dataSize)
	if dataSize <= 0 || err != nil {
		return err
	}

	if req.Command == "append" || req.Command == "prepend" {
		return nil
	}

	siteName := req.Args[0]
	var site *SiteConf
	for _, st := range h.Conf.Sites {
		if st.Name == siteName {
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
