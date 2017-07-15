package app

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Default values.
var interval = 5

// Application object
type Env struct {
	Conf Config
}

// Configuration object.
type Config struct {
	DatabaseFile string   `yaml:"database_file"`
	Interval     int      `yaml:"interval"`
	Services     Services `yaml:"services"`
}

// List of multiple services.
type Services []Service

// Service object, stores details required for tetsing a service.
type Service struct {
	Name    string  `yaml:"name"`
	Url     string  `yaml:"url"`
	Method  string  `yaml:"method"`
	Body    string  `yaml:"body"`
	Headers Headers `yaml:"headers"`
}

// List of multiple services.
type Headers []Header

type Header struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func NewEnv(confFile string) (*Env, error) {
	env := Env{}
	conf := Config{}

	// Find the specified config file
	data, fileErr := ioutil.ReadFile(confFile)
	if fileErr != nil {
		return nil, fileErr
	}

	// Get config from a yaml file
	yamlErr := yaml.Unmarshal([]byte(data), &conf)
	if yamlErr != nil {
		return nil, yamlErr
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
		return nil, errors.New("No services to monitor.")
	}

	// Add config to app object
	env.Conf = conf

	return &env, nil
}

func (env *Env) SetupDb() error {
	db, err := env.ConnectDb()

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

func (env *Env) ConnectDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", env.Conf.DatabaseFile)

	if err != nil {
		return nil, err
	}

	return db, nil
}
