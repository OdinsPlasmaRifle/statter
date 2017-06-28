package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
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

// Application object
type statter struct {
	Conf config
}

// Configuration object.
type config struct {
	DatabaseFile string   `yaml:"database_file"`
	Interval     int      `yaml:"interval"`
	Services     services `yaml:"services"`
}

func main() {
	log.Println("Loading...")

	flag.Parse()

	app := statter{}
	err := app.loadConf(*confFile)

	if err != nil {
		log.Fatalf("Unable to load configuration! Error: %v", err)
	}

	err = app.setupDb()
	if err != nil {
		log.Fatalf("Unable to setup database! Error: %v", err)
	}

	log.Printf("Serving Statter on: %s", *port)

	if *serve == "true" {
		go app.serve(*port)
	}

	log.Println("Enabling monitoring mode.")
	app.monitor()
}

type dbResponses []dbResponse

type dbResponse struct {
	Id         string `db:"id"`
	Name       string `db:"name"`
	Url        string `db:"url"`
	StatusCode int    `db:"status_code"`
	Body       string `db:"body"`
	Created    string `db:"created"`
}

func (app statter) serve(port string) {
	router := http.NewServeMux()
	router.HandleFunc("/services/", app.servicesHandler)

	err := http.ListenAndServe(":"+port, router)
	if err != nil {
		log.Fatal(err)
	}
}

func (app statter) servicesHandler(w http.ResponseWriter, r *http.Request) {
	//w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Get db connection
	db, err := app.connectDb()

	if err != nil {
		// Change to return a response error
		panic(err)
	}

	p := strings.Split(r.URL.Path, "/")

	if len(p) > 3 && len(string(p[2])) > 1 {
		var responses dbResponses

		rows, err := db.Query("SELECT * FROM responses WHERE name=? ORDER BY created DESC LIMIT 100", p[2])

		if err != nil {
			// Change to return a response error
			panic(err)
		}

		defer rows.Close()

		for rows.Next() {
			var r dbResponse

			err = rows.Scan(&r.Id, &r.Name, &r.Url, &r.StatusCode, &r.Body, &r.Created)

			if err != nil {
				// Change to return a response error
				panic(err)
			}

			responses = append(responses, r)
		}

		responseJson, _ := json.MarshalIndent(responses, "", "    ")
		w.Write(responseJson)

	} else {
		// TODO
		// NEED TO OUTPUT SERVICES WITH LIMITED INFO: short name, service label, description
		// NEED TO GET LATEST STATUS FOR EACH SERVICE (OK)
		// LATER ADD AGGREGATES FOR SERVICE UPTIME

		responseJson, _ := json.MarshalIndent(app.Conf.Services, "", "    ")
		w.Write(responseJson)
	}
}

// List of multiple services.
type services []service

// Service object, stores details required for tetsing a service.
// TODO: Add headers
type service struct {
	Name    string  `yaml:"name"`
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

func (app *statter) setupDb() error {
	db, err := app.connectDb()

	if err != nil {
		return err
	}

	// Create tables
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS responses (id INTEGER PRIMARY KEY, name TEXT, url TEXT, status_code INTEGER NULL, body TEXT NULL, created TIMESTAMP DEFAULT CURRENT_TIMESTAMP)")
	_, err = stmt.Exec()

	if err != nil {
		return err
	}

	return nil
}

func (app *statter) connectDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", app.Conf.DatabaseFile)

	if err != nil {
		return nil, err
	}

	return db, nil
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
				go app.iterate()
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
func (app *statter) iterate() {
	monitorTask := make(chan monitorMessage)

	for _, s := range app.Conf.Services {
		go app.test(s, monitorTask)
	}

	for i := 0; i < len(app.Conf.Services); i++ {
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

// Run tests on services and manage errors thrown by said requests. Creates a
// test message channel to catch any errors thrown by the actual request.
func (app *statter) test(s service, monitorTask chan<- monitorMessage) {
	testTask := make(chan testMessage)

	go app.request(s, testTask)
	message := <-testTask

	if message.error != nil {
		monitorTask <- monitorMessage{error: message.error}
	} else {
		// Insert into database
		db, err := app.connectDb()

		if err != nil {
			monitorTask <- monitorMessage{error: err}
			return
		}

		stmt, err := db.Prepare("INSERT INTO responses (name, url, status_code, body) VALUES (?, ?, ?, ?)")

		if err != nil {
			monitorTask <- monitorMessage{error: err}
			return
		}

		body, _ := ioutil.ReadAll(message.data.Body)
		_, err = stmt.Exec(s.Name, s.Url, message.data.StatusCode, string(body))

		if err != nil {
			monitorTask <- monitorMessage{error: err}
		} else {
			log.Println(fmt.Sprintf("Test: %v %v %v", s.Name, s.Url, message.data.Status))
			monitorTask <- monitorMessage{}
		}
	}
}

// Handle the HTTP request on the service. Returns reponse data back to the
// test message.
func (app *statter) request(s service, testTask chan<- testMessage) {
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
