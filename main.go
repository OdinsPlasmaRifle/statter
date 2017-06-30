package main

import (
	"flag"
	"github.com/odinsplasmarifle/statter/app"
	"github.com/odinsplasmarifle/statter/monitor"
	"github.com/odinsplasmarifle/statter/server"
	"log"
	"sync"
)

// CLI Flags.
var confFileFlag = flag.String("config", "", "Config file")
var monitorFlag = flag.String("monitor", "", "Run Statter Monitor")
var serveFlag = flag.String("serve", "", "Serve Statter API")
var portFlag = flag.String("port", "8080", "Statter API port")

func main() {
	log.Println("Loading...")

	flag.Parse()
	env, err := app.NewEnv(*confFileFlag)
	var wg sync.WaitGroup

	if err != nil {
		log.Fatalf("Unable to load configuration! Error: %v\n", err)
	}

	err = env.SetupDb()
	if err != nil {
		log.Fatalf("Unable to setup database! Error: %v\n", err)
	}

	if *serveFlag == "true" {
		log.Printf("Serving Statter on: %s", *portFlag)

		srv := server.Server{env}
		wg.Add(1)
		go srv.Serve(*portFlag)
	}

	if *monitorFlag == "true" {
		log.Println("Enabling monitoring mode.")

		mon := monitor.Monitor{env}
		wg.Add(1)
		go mon.Start()
	}

	wg.Wait()
}
