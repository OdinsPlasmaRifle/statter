package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Set default interval for monitroing in minutes
var monitorInterval = 1

// String slice type for HTTP headers
type stringslice []string

// Setting of stringslice header strings.
func (s *stringslice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Services object for carrying service specific data
type service struct {
	url     string
	method  string
	body    []byte
	headers stringslice
}

// List of services
type services []*service

func main() {
	log.Println("Loading...")

	// Create list of services
	var err error
	ss := services{}
	ss, err = ss.load()

	if err != nil {
		log.Println("Unable to load services!")
	} else {
		log.Println("Successfuly loaded, switching to monitoring mode!")
		ss.monitor()
	}
}

// Load services into a services object
func (ss services) load() (services, error) {
	// Add a service
	service0 := service{}
	service0.url = "https://rehive.com/api/3/asd"
	service0.method = "GET"
	ss = append(ss, &service0)

	return ss, nil
}

// Start up monitoring for sevices, instantiates a set of times to trigger
// off monitroing for services.
func (ss services) monitor() {
	// Create a ticker and fire it off at a set duration
	ticker := time.NewTicker(time.Duration(monitorInterval) * time.Minute)
	quit := make(chan struct{})

	func() {
		for {
			select {
			case <-ticker.C:
				go ss.iterate()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// Message for catching monitoring errors
type monitorMessage struct {
	error error
}

// Iterate through each service and trigger a test job. Creates a monotoring
// channel to catch goroutine errors.
func (ss services) iterate() {
	monitorJob := make(chan monitorMessage)

	for _, s := range ss {
		go s.test(monitorJob)
	}

	for i := 0; i < len(ss); i++ {
		select {
		case message := <-monitorJob:
			if message.error != nil {
				log.Println(fmt.Sprintf("Monitor: %v", message.error))
			}
		}
	}
}

// Message for carrying test repsonse data
type testMessage struct {
	data  *http.Response
	error error
}

// Run tests on services and manage errors thrown by said requests. Creates a
// test message channel to catch any errors thrown by the actual request.
func (s service) test(monitorJob chan<- monitorMessage) {
	testJob := make(chan testMessage)

	go s.request(testJob)
	message := <-testJob

	if message.error != nil {
		monitorJob <- monitorMessage{error: message.error}
	} else {
		log.Println(fmt.Sprintf("Test: %v %v", s.url, message.data.Status))
		monitorJob <- monitorMessage{}
	}
}

// Handle the HTTP request on the service. Returns reponse data back to the
// test message.
func (s *service) request(testJob chan<- testMessage) {
	req, _ := http.NewRequest(s.method, s.url, bytes.NewBuffer(s.body))

	for i := 0; i < len(s.headers); i++ {
		split := strings.Split(s.headers[i], ": ")
		req.Header.Set(split[0], split[1])
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		testJob <- testMessage{data: nil, error: err}
		return
	}
	defer resp.Body.Close()

	testJob <- testMessage{data: resp, error: nil}
}
