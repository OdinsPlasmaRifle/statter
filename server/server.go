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

type dbResponses []dbResponse

type dbResponse struct {
	Id         string `db:"id"`
	Name       string `db:"name"`
	Url        string `db:"url"`
	StatusCode int    `db:"status_code"`
	Body       string `db:"body"`
	Created    string `db:"created"`
}

func (srv Server) servicesHandler(w http.ResponseWriter, r *http.Request) {
	//w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Get db connection
	db, err := srv.ConnectDb()

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

		responseJson, _ := json.MarshalIndent(srv.Conf.Services, "", "    ")
		w.Write(responseJson)
	}
}
