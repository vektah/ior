package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/yhat/wsutil"
)

type Forwarder struct {
	httpForwarder      http.Handler
	websocketForwarder http.Handler
}

func NewForwarder(rpURL *url.URL) *Forwarder {
	shrp := httputil.NewSingleHostReverseProxy(rpURL)
	shrp.FlushInterval = 200 * time.Millisecond
	return &Forwarder{
		httpForwarder:      shrp,
		websocketForwarder: wsutil.NewSingleHostReverseProxy(rpURL),
	}
}

func (f *Forwarder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if wsutil.IsWebSocketRequest(req) {
		f.websocketForwarder.ServeHTTP(w, req)
	} else {
		f.httpForwarder.ServeHTTP(w, req)
	}
}
