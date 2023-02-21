package main

import (
	"log"
	server "mssfoobar/ar2-import/apps/import/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conf := server.Config{}
	err := conf.Load()
	if err != nil {
		log.Fatal("Environment file loading failed.", err)
	}
	importService := server.New()
	importService.Start(conf)

	// Press Ctrl+C to exit the process
	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, os.Interrupt, syscall.SIGTERM)
	<-quitCh
}
