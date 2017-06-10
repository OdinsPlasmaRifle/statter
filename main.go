package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/boltdb/bolt"
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

// Database
var Db *bolt.DB

func main() {
	log.Println("Loading...")

	flag.Parse()
	conf, err := load(*confFile)

	if err != nil {
		log.Fatalf("Unable to load configuration! Error: %v", err)
	}

	Db, err := conf.connect()
	if err != nil {
		log.Fatalf("Unable to connect to database! Error: %v", err)
	}
	defer Db.Close()

	log.Println("Successfuly initiated, switching to monitoring mode!")
	conf.monitor()
}

// Configuration object.
type config struct {
	DatabaseFile string   `yaml:"database_file"`
	Interval     int      `yaml:"interval"`
	Services     services `yaml:"services"`
}

// List of multiple services.
type services []service

// Service object, stores details required for tetsing a service.
// TODO: Add headers
type service struct {
	Url     string  `yaml:"url"`
	Method  string  `yaml:"method"`
	Body    string  `yaml:"body"`
	Headers headers `yaml:"headers"`
}

// List of multiple services.
type headers []header

type header struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
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

	// Set database file
	if conf.DatabaseFile == "" {
		conf.DatabaseFile = "statter.db"
	}

	// Set default interval
	if conf.Interval == 0 {
		conf.Interval = interval
	}

	// Fail if no services are defined
	if len(conf.Services) == 0 {
		return conf, errors.New("No services to monitor.")
	}

	return conf, nil
}

func (conf config) connect() (*bolt.DB, error) {
	db, err := bolt.Open(conf.DatabaseFile, 0666, &bolt.Options{Timeout: 1 * time.Second})

	// Initiate Buckets

	return db, err
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

// Iterate through each service and trigger a test task. Creates a monitoring
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
		// Insert into db with time key and URL bucket
		err := Db.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte(s.Url))
			if err != nil {
				return err
			}

			t := time.Now().Format(time.RFC3339)
			err = bucket.Put([]byte(t), []byte(message.data.Status))
			if err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			monitorTask <- monitorMessage{error: err}
		} else {
			log.Println(fmt.Sprintf("Test: %v %v", s.Url, message.data.Status))
			monitorTask <- monitorMessage{}
		}
	}
}

// Handle the HTTP request on the service. Returns reponse data back to the
// test message.
func (s *service) request(testTask chan<- testMessage) {
	req, _ := http.NewRequest(s.Method, s.Url, bytes.NewBuffer([]byte(s.Body)))

	for _, h := range s.Headers {
		req.Header.Set(h.Name, h.Value)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		testTask <- testMessage{data: nil, error: err}
		return
	}
	defer resp.Body.Close()

	testTask <- testMessage{data: resp, error: nil}
}
