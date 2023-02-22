package main

import (
	"flag"
	server "mssfoobar/ar2-import/apps/import/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var confFile string
	flag.StringVar(&confFile, "c", "../../.env", "environment file")
	flag.Parse()

	if confFile == "" {
		flag.PrintDefaults()
		return
	}

	conf := server.Config{}
	conf.Load(confFile)

	importService := server.New()
	importService.Start(conf)

	// Press Ctrl+C to exit the process
	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, os.Interrupt, syscall.SIGTERM)
	<-quitCh
}
