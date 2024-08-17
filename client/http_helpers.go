package client

import "net/http"

func respondWith500(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(500)
	w.Write([]byte("Internal server error"))
}
