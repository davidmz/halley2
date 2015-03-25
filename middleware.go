package main

import (
	"fmt"
	"net/http"

	"github.com/davidmz/logg"
	"github.com/gorilla/context"
)

type SiteNameChecker struct {
	Log  *logg.Logger `inject:""`
	Conf *Conf        `inject:""`
}

func (s *SiteNameChecker) Check(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var site *SiteConf
		siteName := r.URL.Query().Get("site")
		for _, st := range s.Conf.Sites {
			if st.Name == siteName {
				site = st
				break
			}
		}

		if site == nil {
			s.Log.DEBUG("Unknown site %q (%q)", siteName, r.RequestURI)
			sendJSON(w, http.StatusBadRequest, &PostResponse{
				Status:  statusErr,
				Message: fmt.Sprintf("Unknown site %q", siteName),
			})
			return
		}

		s.Log.TRACE("Site %q", site.Name)

		context.Set(r, "site", site)

		h.ServeHTTP(w, r)
	})
}
