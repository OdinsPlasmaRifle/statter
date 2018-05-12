package monitor

import (
	"bytes"
	"fmt"
	"github.com/odinsplasmarifle/statter/app"
	"gopkg.in/guregu/null.v3"
	"log"
	"net/http"
	"strings"
	"time"
)

type Monitor struct {
	*app.Env
}

// Start up monitoring for sevices, instantiates a ticker to trigger
// off monitoring for services.
func (mon *Monitor) Start() {
	for _, s := range mon.Conf.Services {
		go mon.ticker(s)
	}
}

func (mon *Monitor) ticker(s app.Service) {
	// Create a ticker and fire it off at a set duration.
	ticker := time.NewTicker(time.Duration(s.Interval) * time.Second)
	quit := make(chan struct{})

	func() {
		for {
			select {
			case <-ticker.C:
				go mon.test(s)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// Run tests on services and manage errors thrown by said requests. Creates a
// test message channel to catch any errors thrown by the actual request.
func (mon *Monitor) test(s app.Service) {
	var requestErrorString null.String

	statusCode, requestError := mon.request(s)

	if requestError != nil {
		requestErrorString.String = requestError.Error()
	}

	db, err := mon.ConnectDb()

	if err != nil {
		log.Println(err)
		return
	}

	rowMap := map[string]interface{}{
		"name":       s.Name,
		"url":        s.Url,
		"statusCode": statusCode,
		"error":      requestErrorString,
	}

	rows, err := db.NamedQuery("INSERT INTO responses (name, url, status_code, error) VALUES (:name, :url, :statusCode, :error)", rowMap)

	if err != nil {
		log.Println(err)
		return
	}

	if requestError != nil {
		log.Println(requestErrorString)
	} else {
		log.Println(fmt.Sprintf("%v %v: %v", strings.Title(strings.ToLower(s.Method)), s.Url, statusCode))
	}
}

// Handle the HTTP request on the service. Returns reponse data back to the
// test message.
func (mon *Monitor) request(s app.Service) (int, error) {
	req, _ := http.NewRequest(s.Method, s.Url, bytes.NewBuffer([]byte(s.Body)))

	for _, h := range s.Headers {
		req.Header.Set(h.Name, h.Value)
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
