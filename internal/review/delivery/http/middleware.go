package http

import nethttp "net/http"

func WithCORS(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == nethttp.MethodOptions {
			w.WriteHeader(nethttp.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
