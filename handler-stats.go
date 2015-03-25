package main

import (
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/davidmz/logg"
)

var startTime = time.Now()

type HandlerStats struct {
	Log  *logg.Logger `inject:""`
	Conf *Conf        `inject:""`
}

func (h *HandlerStats) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)

	sendOK(w, &struct {
		Uptime   string `json:"uptime"`
		Memory   uint64 `json:"memory"`
		Sessions uint32 `json:"sessions"`
	}{
		time.Since(startTime).String(),
		mem.Alloc,
		atomic.LoadUint32(&h.Conf.NSessions),
	})
}
