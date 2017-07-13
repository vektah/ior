package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/yhat/wsutil"
)

type Forwarder struct {
	httpForwarder      http.Handler
	websocketForwarder http.Handler
}

func NewForwarder(rpURL *url.URL) *Forwarder {
	return &Forwarder{
		httpForwarder:      httputil.NewSingleHostReverseProxy(rpURL),
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
