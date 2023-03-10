package server

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"logur.dev/logur"
)

func (srv *ImportService) routes(logger logur.KVLogger) http.Handler {
	logware := func(next httprouter.Handle, endpoint string) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			logger.Debug("Endpoint called: " + endpoint)
			next(w, r, ps)
		}
	}
	router := httprouter.New()
	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
		log.Println(r)
	})
	router.HandleOPTIONS = true
	// Set handler
	router.GET("/import-file/:uploadType/:requestId", logware(srv.getPresignedUrl, "getPresignedUrl"))
	router.GET("/upload/bucket/:bucket/expire/:expire/objectName/*objectName", logware(srv.upload, "upload"))
	router.GET("/download/bucket/:bucket/expire/:expire/objectName/*objectName", logware(srv.download, "download"))
	router.GET("/info/liveness", logware(srv.getLiveness, "getLiveness"))
	router.GET("/info/readiness", logware(srv.getReadiness, "getReadiness"))
	return router
}
