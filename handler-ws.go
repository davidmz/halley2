package main

import (
	"net/http"
	"time"

	"github.com/davidmz/halley2/internal/npool"
	"github.com/davidmz/logg"
	"github.com/facebookgo/inject"
	"github.com/gorilla/context"
	"github.com/gorilla/websocket"
)

type HandlerWs struct {
	Log  *logg.Logger        `inject:""`
	Upgr *websocket.Upgrader `inject:""`
	Pool npool.NamedPool     `inject:""`
	Conf *Conf               `inject:""`
}

const (
	PingInterval     = time.Second * 60
	KeepAliveTimeout = PingInterval * 2
)

func (h *HandlerWs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.Log.ChildWithPrefix(r.RemoteAddr)

	log.TRACE("New ws request")
	defer log.TRACE("End of ws request")

	endHandling := make(chan struct{})
	defer close(endHandling)

	site := context.Get(r, "site").(*SiteConf)

	conn, err := h.Upgr.Upgrade(w, r, nil)
	if err != nil {
		log.DEBUG("Can not upgrade: %v", err)
		return
	}
	defer conn.Close()

	KeepAlive(conn, endHandling, log)

	sess := NewSession(conn)
	if err := inject.Populate(log, site, h.Pool, h.Conf, sess); err != nil {
		log.ERROR("Initialization error: %v", err)
		return
	}

	sess.Run()
}

func KeepAlive(conn *websocket.Conn, close <-chan struct{}, log *logg.Logger) {
	go func() {
		ticker := time.NewTicker(PingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.TRACE("Ping")
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Time{}); err != nil {
					log.DEBUG("Ping error %v", err)
					return
				}
			case <-close:
				conn.SetPongHandler(nil)
				return
			}
		}
	}()

	conn.SetReadDeadline(time.Now().Add(KeepAliveTimeout))

	conn.SetPongHandler(func(s string) error {
		log.TRACE("Pong %q", s)
		conn.SetReadDeadline(time.Now().Add(KeepAliveTimeout))
		return nil
	})
}
