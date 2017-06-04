package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Default values.
var interval = 60

// CLI Flags.
var confFile = flag.String("config", "", "Config file")

func main() {
	log.Println("Loading...")

	flag.Parse()
	conf, err := load(*confFile)

	if err != nil {
		log.Fatalf("Unable to load services! Error: %v", err)
	} else {
		log.Println("Successfuly loaded, switching to monitoring mode!")
		conf.monitor()
	}
}

// Configuration object.
type config struct {
	Interval int      `yaml:"interval"`
	Services services `yaml:"services"`
}

// List of multiple services.
type services []service

// Service object, stores details required for tetsing a service.
// TODO: Add headers
type service struct {
	Url    string `yaml:"url"`
	Method string `yaml:"method"`
	Body   []byte `yaml:"body"`
}

// Load configuration.
func load(confFile string) (config, error) {
	conf := config{}

	// Find the specified config file
	data, fileErr := ioutil.ReadFile(confFile)
	if fileErr != nil {
		return conf, fileErr
	}

	// Get config from a yaml file
	yamlErr := yaml.Unmarshal([]byte(data), &conf)
	if yamlErr != nil {
		return conf, yamlErr
	}

	// Set default interval
	if conf.Interval == 0 {
		conf.Interval = interval
	}

	// Faile if no services are defined
	if len(conf.Services) == 0 {
		return conf, errors.New("No services to monitor.")
	}

	return conf, nil
}

// Start up monitoring for sevices, instantiates a set of times to trigger
// off monitoring for services.
func (conf config) monitor() {
	// Create a ticker and fire it off at a set duration.
	ticker := time.NewTicker(time.Duration(conf.Interval) * time.Second)
	quit := make(chan struct{})

	func() {
		for {
			select {
			case <-ticker.C:
				go conf.Services.iterate()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// Message for catching monitoring errors.
type monitorMessage struct {
	error error
}

// Iterate through each service and trigger a test task. Creates a monotoring
// channel to catch goroutine errors.
func (ss services) iterate() {
	monitorTask := make(chan monitorMessage)

	for _, s := range ss {
		go s.test(monitorTask)
	}

	for i := 0; i < len(ss); i++ {
		select {
		case message := <-monitorTask:
			if message.error != nil {
				log.Println(fmt.Sprintf("Monitor: %v", message.error))
			}
		}
	}
}

// Message for carrying test repsonse data.
type testMessage struct {
	data  *http.Response
	error error
}

// Run tests on services and manage errors thrown by said requests. Creates a
// test message channel to catch any errors thrown by the actual request.
func (s service) test(monitorTask chan<- monitorMessage) {
	testTask := make(chan testMessage)

	go s.request(testTask)
	message := <-testTask

	if message.error != nil {
		monitorTask <- monitorMessage{error: message.error}
	} else {
		log.Println(fmt.Sprintf("Test: %v %v", s.Url, message.data.Status))
		monitorTask <- monitorMessage{}
	}
}

// Handle the HTTP request on the service. Returns reponse data back to the
// test message.
func (s *service) request(testTask chan<- testMessage) {
	req, _ := http.NewRequest(s.Method, s.Url, bytes.NewBuffer(s.Body))

	// TODO: Set headers

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		testTask <- testMessage{data: nil, error: err}
		return
	}
	defer resp.Body.Close()

	testTask <- testMessage{data: resp, error: nil}
}
