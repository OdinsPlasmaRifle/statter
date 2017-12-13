package server

import (
	"encoding/json"
	"github.com/odinsplasmarifle/statter/app"
	"log"
	"net/http"
	"strings"
)

type Server struct {
	*app.Env
}

func (srv Server) Serve(port string) {
	router := http.NewServeMux()
	router.HandleFunc("/services/", srv.servicesHandler)

	err := http.ListenAndServe(":"+port, router)
	if err != nil {
		log.Fatal(err)
	}
}

// Response struct for data stored in the database.
type response struct {
	Id         string `db:"id" json:"id"`
	Name       string `db:"name" json:"name"`
	Url        string `db:"url" json:"url"`
	StatusCode int    `db:"status_code" json:"statusCode"`
	Created    string `db:"created" json:"created"`
}

// Service struct for custom builtd outputs.
type service struct {
	Name        string      `json:"name"`
	Label       string      `json:"label"`
	Description string      `json:"description"`
	Responses   []*response `json:"responses"`
}

func (srv Server) servicesHandler(w http.ResponseWriter, r *http.Request) {
	//w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Get db connection
	db, err := srv.ConnectDb()

	if err != nil {
		panic(err)
	}

	p := strings.Split(r.URL.Path, "/")

	if len(p) > 3 && len(string(p[2])) > 1 {
		rs := []*response{}
		err := db.Select(&rs, "SELECT * FROM responses WHERE name=$1", p[2])

		if err != nil {
			panic(err)
		}

		responseJson, _ := json.MarshalIndent(rs, "", "    ")
		w.Write(responseJson)

	} else {
		var ss []service

		for _, confS := range srv.Conf.Services {
			rs := []*response{}
			err := db.Select(&rs, "SELECT * FROM responses WHERE name=$1", confS.Name)

			if err != nil {
				panic(err)
			}

			s := service{Name: confS.Name, Label: confS.Label, Description: confS.Description, Responses: rs}
			ss = append(ss, s)
		}

		responseJson, _ := json.MarshalIndent(ss, "", "    ")
		w.Write(responseJson)
	}
}
