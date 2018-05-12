package server

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/odinsplasmarifle/statter/app"
	"gopkg.in/guregu/null.v3"
	"log"
	"net/http"
)

type Server struct {
	*app.Env
}

func (srv Server) Serve() {
	router := httprouter.New()

	router.GET("/services/", srv.listServices)
	router.GET("/responses/", srv.listResponses)

	log.Fatal(http.ListenAndServe(":"+srv.Conf.Port, router))
}

// Service struct for custom built outputs.
type service struct {
	Name                  string    `json:"name"`
	Label                 string    `json:"label"`
	Description           string    `json:"description"`
	TotalRequests         null.Int  `json:"totalRequests" db:"total"`
	TotalFailedRequests   null.Int  `json:"totalFailedRequests" db:"total_failed"`
	LastFailedRequestDate null.Time `json:"lastFailedRequestDate" db:"last_failed"`
	StatusCode            null.Int  `json:"statusCode" db:"status_code"`
}

// Response struct for data stored in the database.
type response struct {
	Id         string      `db:"id" json:"id"`
	Name       string      `db:"name" json:"name"`
	Url        string      `db:"url" json:"url"`
	StatusCode int         `db:"status_code" json:"statusCode"`
	Error      null.String `db:"error" json:"error"`
	Created    null.Time   `db:"created" json:"created"`
}

// List services and filter services by name.
func (srv Server) listServices(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	db, err := srv.ConnectDb()

	if err != nil {
		panic(err)
	}

	var ss []service
	filters := r.URL.Query()

	for _, confService := range srv.Conf.Services {
		mustAppend := true

		if filters["name"] != nil {
			if filters["name"][0] != "" && confService.Name == filters["name"][0] {
				mustAppend = true
			} else {
				mustAppend = false
			}
		}

		if mustAppend {
			s := service{}
			err := db.Get(&s,
				`SELECT status_code, count(*) AS total,
					(SELECT count(*) FROM responses tf
						WHERE tf.name=$1 AND (tf.status_code >= 300 OR tf.status_code < 200)) AS total_failed,
					(SELECT created FROM responses tf2
						WHERE tf2.name=$1 AND (tf2.status_code >= 300 OR tf2.status_code < 200) ORDER BY id DESC LIMIT 1) AS last_failed
				 FROM responses WHERE name=$1 ORDER BY id DESC`, confService.Name)

			if err != nil {
				panic(err)
			}

			s.Name = confService.Name
			s.Label = confService.Label
			s.Description = confService.Description
			ss = append(ss, s)
		}
	}

	responseJson, _ := json.MarshalIndent(ss, "", "    ")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)
}

// List services and filter responses by service name.
func (srv Server) listResponses(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	db, err := srv.ConnectDb()

	if err != nil {
		panic(err)
	}

	rs := []*response{}
	filters := r.URL.Query()

	if filters["name"] != nil && filters["name"][0] != "" {
		err = db.Select(&rs, "SELECT * FROM responses WHERE name=$1 ORDER BY id DESC LIMIT 100", filters["name"][0])
	} else {
		err = db.Select(&rs, "SELECT * FROM responses ORDER BY id DESC LIMIT 100")
	}

	if err != nil {
		panic(err)
	}

	responseJson, _ := json.MarshalIndent(rs, "", "    ")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJson)
}
