package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/davidmz/halley2/internal/channel"
	"github.com/davidmz/halley2/internal/npool"
	"github.com/davidmz/logg"
	"github.com/gorilla/context"
)

type HandlerPost struct {
	Conf *Conf           `inject:""`
	Log  *logg.Logger    `inject:""`
	Pool npool.NamedPool `inject:""`
}

type PostRequest struct {
	ChanName    string          `json:"channel"`
	Token       []byte          `json:"token"`
	Signature   []byte          `json:"auth"`
	MessageBody json.RawMessage `json:"message"`
}

const (
	statusOK  = "ok"
	statusErr = "error"
)

type PostResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (h *HandlerPost) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.Log.ChildWithPrefix("POST from " + r.RemoteAddr)

	log.TRACE("New post request")
	defer log.TRACE("End of post request")

	defer r.Body.Close()

	site := context.Get(r, "site").(*SiteConf)

	req := &PostRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		log.DEBUG("Cannot decode json: %v", err)
		sendJSON(w, http.StatusBadRequest, &PostResponse{
			Status:  statusErr,
			Message: fmt.Sprintf("Cannot decode json: %v", err),
		})
		return
	}

	if !CheckToken(req.Token, h.Conf.Secret) {
		log.DEBUG("Invalid token: %v", req.Token)
		w.WriteHeader(http.StatusBadRequest)
		sendJSON(w, http.StatusBadRequest, &PostResponse{
			Status:  statusErr,
			Message: fmt.Sprintf("Invalid token"),
		})
		return
	}

	mac := hmac.New(sha256.New, site.PostSecret)
	mac.Write(req.Token)
	sign := mac.Sum(nil)

	if !hmac.Equal(sign, req.Signature) {
		log.DEBUG("Invalid signature: %v", req.Token)
		sendJSON(w, http.StatusBadRequest, &PostResponse{
			Status:  statusErr,
			Message: fmt.Sprintf("Invalid signature"),
		})
		return
	}

	ch := h.Pool.Get(channel.ChanKey{
		Site: site.Name,
		Name: req.ChanName,
	}).(*channel.Channel)

	ch.AddMessage(req.MessageBody)

	sendOK(w, &PostResponse{Status: statusOK})
}
