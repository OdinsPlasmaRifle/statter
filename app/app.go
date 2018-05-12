package app

import (
	"errors"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Default values.
var interval = 60

// Environment object
type Env struct {
	Conf Config
}

// Configuration object.
type Config struct {
	Database string    `yaml:"database"`
	Port     string    `yaml:"port"`
	Services []Service `yaml:"services"`
}

// Service object, stores details required for monitoring a service.
type Service struct {
	Name        string   `yaml:"name"`
	Label       string   `yaml:"label"`
	Description string   `yaml:"description"`
	Url         string   `yaml:"url"`
	Method      string   `yaml:"method"`
	Body        string   `yaml:"body"`
	Headers     []Header `yaml:"headers"`
	Interval    int      `yaml:"interval"`
}

// Headers object, stores header details.
type Header struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func NewEnv(confFile string) (*Env, error) {
	env := Env{}
	conf := Config{}

	// Find the specified config file.
	data, fileErr := ioutil.ReadFile(confFile)
	if fileErr != nil {
		return nil, fileErr
	}

	// Get config from a yaml file.
	yamlErr := yaml.Unmarshal([]byte(data), &conf)
	if yamlErr != nil {
		return nil, yamlErr
	}

	// Set database file.
	if conf.Database == "" {
		conf.Database = "statter.db"
	}

	// Set port.
	if conf.Port == "" {
		conf.Port = "8080"
	}

	// Fail if no services are defined.
	if len(conf.Services) == 0 {
		return nil, errors.New("No services to monitor.")
	}

	// Set default interval for each service.
	for i := 0; i < len(conf.Services); i++ {
		s := &conf.Services[i]

		if s.Interval == 0 {
			s.Interval = interval
		}
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

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS responses (id INTEGER PRIMARY KEY, name TEXT, url TEXT, status_code INTEGER NULL, error TEXT NULL, created TIMESTAMP DEFAULT CURRENT_TIMESTAMP)")

	if err != nil {
		return err
	}

	return nil
}

func (env *Env) ConnectDb() (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", env.Conf.Database+"?_busy_timeout=5000&parseTime=true")

	if err != nil {
		return nil, err
	}

	return db, nil
}
