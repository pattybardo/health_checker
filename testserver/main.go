package main

import (
	"log"
	"log/slog"
	"math/rand"
	"net/http"
)

var failCode int = 503
var port string = "8941"

func main() {
	slog.Info("starting flaky-healthz")

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if rand.Intn(100) < 25 {
			slog.Warn("returning failure", "code", failCode)
			w.WriteHeader(failCode)
			return
		}
		w.WriteHeader(http.StatusOK)
		slog.Info("OK")
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
