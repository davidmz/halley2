package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/davidmz/halley2/internal/channel"
	"github.com/davidmz/halley2/internal/npool"
	"github.com/davidmz/logg"
	"github.com/davidmz/memcache"
	"github.com/facebookgo/inject"
	"github.com/gorilla/websocket"
	"github.com/sqs/mux"
)

func main() {
	conf, err := ReadConf()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Config error:", err)
		os.Exit(1)
	}

	log := logg.New(conf.LogLevel, logg.DefaultWriter)

	handlerWs := new(HandlerWs)
	handlerPost := new(HandlerPost)
	handlerToken := new(HandlerToken)
	handlerStats := new(HandlerStats)
	handlerMemc := new(HandlerMemc)
	siteNameChecker := new(SiteNameChecker)

	wsUgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true }, // TODO CORS
	}

	chanPool := npool.New(
		(*channel.Channel)(nil),
		&channel.ChanConf{
			RingSize: conf.ChannelSize,
			TTL:      conf.MsgLifetime,
		},
	)

	if err := inject.Populate(
		log, wsUgrader, chanPool,
		handlerWs, handlerPost, handlerToken, handlerStats, handlerMemc,
		conf, siteNameChecker,
	); err != nil {
		log.FATAL("Initialization error: %v", err)
		os.Exit(1)
	}

	router := mux.NewRouter()
	router.Handle("/ws", siteNameChecker.Check(handlerWs))
	router.Handle("/post", siteNameChecker.Check(handlerPost))
	router.Handle("/token", siteNameChecker.Check(handlerToken))
	router.Handle("/stats", handlerStats)

	if conf.ListenMemc != "" {
		log.INFO("Starting memcache server at %v", conf.ListenMemc)
		ln, err := net.Listen("tcp", conf.ListenMemc)
		if err != nil {
			log.FATAL("Can not start memcache server: %v", err)
			os.Exit(1)
		}
		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					log.ERROR("Memcache listen error: %v", err)
					continue
				}

				go memcache.HandleConnection(conn, handlerMemc)
			}
		}()
	}

	log.INFO("Starting server at %v", conf.ListenAddr)
	if err := http.ListenAndServe(conf.ListenAddr, router); err != nil {
		log.FATAL("Can not start server: %v", err)
		os.Exit(1)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		log.Println(r)
		return true
	},
}

func sendJSON(resp http.ResponseWriter, status int, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(map[string]string{"error": err.Error()})
		status = http.StatusInternalServerError
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1
	resp.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0
	resp.Header().Set("Expires", "0")                                         // Proxies
	resp.WriteHeader(status)
	resp.Write(b)
}

func sendError(resp http.ResponseWriter, status int, message string) {
	sendJSON(resp, status, map[string]string{"error": message})
}

func sendOK(resp http.ResponseWriter, v interface{}) {
	sendJSON(resp, http.StatusOK, v)
}
