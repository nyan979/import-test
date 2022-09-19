package main

import (
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

// does not set cors origin for development
func corsware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		corsOrigin := os.Getenv("CORS_ALLOW_ORIGIN")

		if len(corsOrigin) > 0 {
			w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		}

		next(w, r, ps)
	}
}

func (app *Application) routes() http.Handler {
	router := httprouter.New()

	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Origin", r.URL.Host)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
	})

	router.HandleOPTIONS = true

	// Set handler
	router.Handle(http.MethodGet, "/import-file/:uploadType/:requestId", corsware(app.getPresignedUrl))
	router.Handle(http.MethodGet, "/info/health", corsware(app.getHealthInfo))
	router.Handle(http.MethodGet, "/info/version", corsware(app.getVersionInfo))

	return router
}
