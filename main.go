package main

import (
	"net/http"
)

func healthcheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("./"))))
	mux.HandleFunc("/healthz", healthcheck)
	server := http.Server{Handler: mux, Addr: ":8080"}
	server.ListenAndServe()
}
