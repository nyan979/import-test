package main

import (
	"fmt"
	"net/http"
)

func getPresignedUrl(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "")
}

func SetupRoutes() {
	http.HandleFunc("/getUrl", getPresignedUrl)
	http.ListenAndServe(":5000", nil)
}
