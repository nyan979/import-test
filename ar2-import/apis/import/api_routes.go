package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *Application) routes() http.Handler {
	router := httprouter.New()

	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		//w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
		log.Println(r)
	})

	router.HandleOPTIONS = true

	// Set handler
	router.GET("/import-file/:uploadType/:requestId", app.getPresignedUrl)
	router.GET("/info/health", app.getHealthInfo)
	router.GET("/info/version", app.getVersionInfo)

	return router
}
