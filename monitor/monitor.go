package monitor

import (
	"bytes"
	"fmt"
	"github.com/odinsplasmarifle/statter/app"
	"log"
	"net/http"
	"time"
)

type Monitor struct {
	*app.Env
}

// Start up monitoring for sevices, instantiates a ticker to trigger
// off monitoring for services.
func (mon *Monitor) Start() {
	// Create a ticker and fire it off at a set duration.
	ticker := time.NewTicker(time.Duration(mon.Conf.Interval) * time.Second)
	quit := make(chan struct{})

	func() {
		for {
			select {
			case <-ticker.C:
				go mon.iterate()
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

// Iterate through each service and trigger a test task. Creates a monitoring
// channel to catch goroutine errors.
func (mon *Monitor) iterate() {
	monitorTask := make(chan monitorMessage)

	for _, s := range mon.Conf.Services {
		go mon.test(s, monitorTask)
	}

	for i := 0; i < len(mon.Conf.Services); i++ {
		select {
		case message := <-monitorTask:
			if message.error != nil {
				log.Println(fmt.Sprintf("Monitor: %v", message.error))
			}
		}
	}
}

// Message for carrying test response data.
type testMessage struct {
	statusCode int
	error      error
}

// Run tests on services and manage errors thrown by said requests. Creates a
// test message channel to catch any errors thrown by the actual request.
func (mon *Monitor) test(s app.Service, monitorTask chan<- monitorMessage) {
	testTask := make(chan testMessage)

	go mon.request(s, testTask)
	message := <-testTask

	if message.error != nil {
		monitorTask <- monitorMessage{error: message.error}
	} else {
		db, err := mon.ConnectDb()

		if err != nil {
			monitorTask <- monitorMessage{error: err}
			return
		}

		_, err = db.Exec("INSERT INTO responses (name, url, status_code) VALUES (?, ?, ?)", s.Name, s.Url, message.statusCode)

		if err != nil {
			monitorTask <- monitorMessage{error: err}
		} else {
			log.Println(fmt.Sprintf("Test: %v %v %v", s.Name, s.Url, message.statusCode))
			monitorTask <- monitorMessage{}
		}
	}
}

// Handle the HTTP request on the service. Returns reponse data back to the
// test message.
func (mon *Monitor) request(s app.Service, testTask chan<- testMessage) {
	req, _ := http.NewRequest(s.Method, s.Url, bytes.NewBuffer([]byte(s.Body)))

	for _, h := range s.Headers {
		req.Header.Set(h.Name, h.Value)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		testTask <- testMessage{statusCode: 0, error: err}
		return
	}
	defer resp.Body.Close()

	testTask <- testMessage{statusCode: resp.StatusCode, error: nil}
}
