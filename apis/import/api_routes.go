package main

import (
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

func corsware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		corsOrigin := os.Getenv("CORS_ALLOW_ORIGIN")

		if len(corsOrigin) > 0 {
			w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		}

		next(w, r, ps)
	}
}

func (app *Config) routes() http.Handler {
	router := httprouter.New()

	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		header := w.Header()
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Methods", "POST, OPTIONS")

		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
	})

	router.HandleOPTIONS = true

	router.Handle(http.MethodPost, "/import", corsware(app.getPresignedUrl))

	return router
}
