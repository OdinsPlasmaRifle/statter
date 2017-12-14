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
var confFileFlag = flag.String("config", "conf.yaml", "Config file")
var monitorFlag = flag.String("monitor", "true", "Run Statter Monitor")
var serveFlag = flag.String("serve", "true", "Serve Statter API")

func main() {
	log.Println("Loading...")

	flag.Parse()
	env, err := app.NewEnv(*confFileFlag)

	if err != nil {
		log.Fatalf("Unable to load configuration! Error: %v\n", err)
	}

	err = env.SetupDb()
	if err != nil {
		log.Fatalf("Unable to setup database! Error: %v\n", err)
	}

	var wg sync.WaitGroup

	if *serveFlag == "true" {
		log.Printf("Serving Statter on: %s", env.Conf.Port)

		srv := server.Server{env}
		wg.Add(1)
		go srv.Serve()
	}

	if *monitorFlag == "true" {
		log.Println("Starting monitoring mode.")

		mon := monitor.Monitor{env}
		wg.Add(1)
		go mon.Start()
	}

	wg.Wait()
}
