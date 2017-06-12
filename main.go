package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/boltdb/bolt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

// Default values.
var interval = 60

// CLI Flags.
var confFile = flag.String("config", "", "Config file")
var serve = flag.String("serve", "", "Serve Statter API")
var port = flag.String("port", "8080", "Statter API port")

// Database
var Db *bolt.DB

func main() {
	log.Println("Loading...")

	flag.Parse()

	app := statter{}
	err := app.loadConf(*confFile)

	if err != nil {
		log.Fatalf("Unable to load configuration! Error: %v", err)
	}

	err = app.connectDb()
	if err != nil {
		log.Fatalf("Unable to connect to database! Error: %v", err)
	}

	if *serve == "true" {
		log.Printf("Serving Statter on: %s", *port)
		go app.serve(*port)
	}

	log.Println("Enabling monitoring mode.")
	app.monitor()
}

func (app statter) serve(port string) {
	router := http.NewServeMux()
	router.HandleFunc("/services/", app.servicesHandler)

	err := http.ListenAndServe(":"+port, router)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully served statter API")
}

func (app statter) servicesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	p := strings.Split(r.URL.Path, "/")

	if len(p) > 1 && len(string(p[0])) > 1 {
		responseJson, _ := json.MarshalIndent(app.Conf.Services, "", "    ")
		w.Write(responseJson)
	} else {
		responseJson, _ := json.MarshalIndent(app.Conf.Services, "", "    ")
		w.Write(responseJson)
	}
}

// Application object
type statter struct {
	Conf config
	Db   *bolt.DB
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
func (app *statter) loadConf(confFile string) error {
	conf := config{}

	// Find the specified config file
	data, fileErr := ioutil.ReadFile(confFile)
	if fileErr != nil {
		return fileErr
	}

	// Get config from a yaml file
	yamlErr := yaml.Unmarshal([]byte(data), &conf)
	if yamlErr != nil {
		return yamlErr
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
		return errors.New("No services to monitor.")
	}

	// Add config to app object
	app.Conf = conf

	return nil
}

func (app *statter) connectDb() error {
	db, err := bolt.Open(app.Conf.DatabaseFile, 0666, &bolt.Options{Timeout: 1 * time.Second})

	if err != nil {
		return err
	}

	// Initiate Buckets
	err = db.Update(func(tx *bolt.Tx) error {
		for _, s := range app.Conf.Services {
			_, err := tx.CreateBucketIfNotExists([]byte(s.Url))
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Add database to app object
	app.Db = db

	return nil
}

// Start up monitoring for sevices, instantiates a set of times to trigger
// off monitoring for services.
func (app *statter) monitor() {
	// Create a ticker and fire it off at a set duration.
	ticker := time.NewTicker(time.Duration(app.Conf.Interval) * time.Second)
	quit := make(chan struct{})

	func() {
		for {
			select {
			case <-ticker.C:
				go app.Conf.Services.iterate(app.Db)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	defer app.Db.Close()
}

// Message for catching monitoring errors.
type monitorMessage struct {
	error error
}

// Iterate through each service and trigger a test task. Creates a monitoring
// channel to catch goroutine errors.
func (ss services) iterate(db *bolt.DB) {
	monitorTask := make(chan monitorMessage)

	for _, s := range ss {
		go s.test(db, monitorTask)
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

// Message for carrying test response data.
type testMessage struct {
	data  *http.Response
	error error
}

type httpResponse struct {
	StatusCode int
	Body       []byte
}

type httpResponses []httpResponse

// Run tests on services and manage errors thrown by said requests. Creates a
// test message channel to catch any errors thrown by the actual request.
func (s service) test(db *bolt.DB, monitorTask chan<- monitorMessage) {
	testTask := make(chan testMessage)

	go s.request(testTask)
	message := <-testTask

	if message.error != nil {
		monitorTask <- monitorMessage{error: message.error}
	} else {
		responseObject := httpResponse{}
		responseObject.StatusCode = message.data.StatusCode
		responseObject.Body, _ = ioutil.ReadAll(message.data.Body)
		responseJson, err := json.MarshalIndent(responseObject, "", "    ")

		if err != nil {
			monitorTask <- monitorMessage{error: err}
			return
		}

		// Insert into db with time key and URL bucket
		err = db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(s.Url))

			t := time.Now().Format(time.RFC3339)
			err := bucket.Put([]byte(t), responseJson)
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
