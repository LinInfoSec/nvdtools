package main

import (
	"log"
	"net/http"
	"time"
	"encoding/json"
	"io/ioutil"
	"context"
	"strconv"

	"database/sql"
	"github.com/pkg/errors"
	"github.com/facebookincubator/flog"
	_ "github.com/go-sql-driver/mysql"
)



func handleMonitor(db *sql.DB, action int) func (http.ResponseWriter,*http.Request){
	if action == ADD {
		return func(w http.ResponseWriter,r *http.Request) {
			log.Println("add")

			if r.Method != "POST" {
				http.Error(w, "Method is not supported.", http.StatusNotFound)
			}

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w,"Missing body",http.StatusBadRequest)
				log.Println(err)
				return
			}

			var conf Configuration
			if err = json.Unmarshal(body,&conf); err != nil {
				http.Error(w, "Incorrect JSON payload",http.StatusBadRequest)
				return
			}

			if err = AddConfiguration(db, conf); err != nil {
				if err == errors.New("Internal error") {
					http.Error(w, "Internal error",http.StatusInternalServerError)
				} else {
					http.Error(w, err.Error(),http.StatusBadRequest)
				}
				return
			}
		}
	} else if action == UPDATE {
		return func(w http.ResponseWriter,r *http.Request) {
			log.Println("update")

			if r.Method != "POST" {
				http.Error(w, "Method is not supported.", http.StatusNotFound)
			}

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Missing body",http.StatusBadRequest)
				log.Println(err)
				return
			}

			var conf Configuration
			if err = json.Unmarshal(body,&conf); err != nil {
				http.Error(w, err.Error(),http.StatusBadRequest)
				log.Println(err)
				return
			}

			if err = UpdateConfiguration(db, conf); err != nil {
				if err == errors.New("Internal error") {
					http.Error(w, "Internal error",http.StatusInternalServerError)
				} else {
					http.Error(w, err.Error(),http.StatusBadRequest)
				}
				return
			}
		}
	} else if action == REMOVE {
		return func(w http.ResponseWriter,r *http.Request) {
			log.Println("remove")

			if r.Method != "POST" {
				http.Error(w, "Method is not supported.", http.StatusNotFound)
			}

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(),http.StatusBadRequest)
				log.Println(err)
				return
			}


			var conf struct {
				Name string `json:"configuration"`
			}


			if err = json.Unmarshal(body,&conf); err != nil {
				http.Error(w, err.Error(),http.StatusBadRequest)
				log.Println(err)
				return
			}

			if err = DeleteConfiguration(db, conf.Name); err != nil {
				if err == errors.New("Internal error") {
					http.Error(w, "Internal error",http.StatusInternalServerError)
				} else {
					http.Error(w, err.Error(),http.StatusBadRequest)
				}
			}


		}
	} else if action == GET {
		return func(w http.ResponseWriter,r *http.Request) {
			log.Println("get")

			if r.Method != "GET" {
				http.Error(w, "Method is not supported.", http.StatusNotFound)
			}
			
			r.ParseForm()
			
			if len(r.Form["name"]) != 1 {
				http.Error(w, "Missing configuration name",http.StatusBadRequest)
				return
			}

			res, err := GetConfiguration(db,r.Form["name"][0])
			if err != nil {
				if err == errors.New("Internal error") {
					http.Error(w, "Internal error",http.StatusInternalServerError)
					return
				} else {
					http.Error(w, err.Error(),http.StatusBadRequest)
					return
				}
			}

			serialized, err := json.Marshal(res)
			if err != nil {
				flog.Error(err)
				http.Error(w, "Internal error",http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(serialized)

		}
	}

	panic("unknown action")
}

func handleSearch(db *sql.DB) func (http.ResponseWriter,*http.Request){
	return func (w http.ResponseWriter,r *http.Request){
		log.Println("Search")
		if r.Method != "GET" {
			http.Error(w, "Method is not supported.", http.StatusNotFound)
			return
		}

		r.ParseForm()
		ctx := context.Background()
		if len(r.Form["query"]) != 1 {
			http.Error(w, "Bad number of queries",http.StatusBadRequest)
			return
		}

		var start int
		if len(r.Form["start"]) != 0 {
			s, err := strconv.Atoi(r.Form["start"][0])
			log.Println(s)
			if err == nil {
				start = s
			}
		}

		count := 20
		if len(r.Form["count"]) != 0 {
			s, err := strconv.Atoi(r.Form["count"][0])
			if err == nil || s > 50 {
				count = s
			}
		}

		data := searchData {
			query: r.Form["query"][0],
			start: start,
			count: count,
		}

		res, err := CPESearch(data,db,ctx)
		if err != nil {
			if err == errors.New("Internal error") {
				http.Error(w, "Internal error",http.StatusInternalServerError)
				return
			} else {
				http.Error(w, err.Error(),http.StatusBadRequest)
				return
			}
		}

		serialized, err := json.Marshal(res)
		if err != nil {
			flog.Error(err)
			http.Error(w, "Internal error",http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(serialized)
	}
}

func handleInfo(db *sql.DB) func (http.ResponseWriter,*http.Request){
	return func (w http.ResponseWriter,r *http.Request){
		log.Println("productVersions")
		if r.Method != "GET" {
			http.Error(w, "Method is not supported.", http.StatusNotFound)
			return
		}

		r.ParseForm()
		ctx := context.Background()
		if len(r.Form["product"]) != 1 {
			http.Error(w, "Missing product",http.StatusBadRequest)
			return
		}
		if len(r.Form["vendor"]) != 1 {
			http.Error(w, "Missing vendor",http.StatusBadRequest)
			return
		}
		if len(r.Form["part"]) != 1 {
			http.Error(w, "Missing part",http.StatusBadRequest)
			return
		}

		var start int
		if len(r.Form["start"]) != 0 {
			s, err := strconv.Atoi(r.Form["start"][0])
			log.Println(s)
			if err == nil {
				start = s
			}
		}

		count := 20
		if len(r.Form["count"]) != 0 {
			s, err := strconv.Atoi(r.Form["count"][0])
			if err == nil || s > 50 {
				count = s
			}
		}

		data := InfoData {
			part: r.Form["part"][0],
			vendor: r.Form["vendor"][0],
			product: r.Form["product"][0],
			start: start,
			count: count,
		}

		res, err := getCpes(data,db,ctx)
		if err != nil {
			if err == errors.New("Internal error") {
				http.Error(w, "Internal error",http.StatusInternalServerError)
				return
			} else {
				http.Error(w, err.Error(),http.StatusBadRequest)
				return
			}
		}

		serialized, err := json.Marshal(res)
		if err != nil {
			flog.Error(err)
			http.Error(w, "Internal error",http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(serialized)
	}
}


func main() {
	LoadConfig()

	db, err := sql.Open("mysql",DB_DSN)
	if err != nil {
		log.Fatal(err)
	}


	http.HandleFunc("/monitor/add", handleMonitor(db,ADD))  // Add configurations to be monitored
	http.HandleFunc("/monitor/remove", handleMonitor(db,REMOVE))  // Remove configurations to be monitored
	http.HandleFunc("/monitor/update", handleMonitor(db,UPDATE))  // Remove configurations to be monitored
	http.HandleFunc("/monitor/get", handleMonitor(db,GET))  // GET the stack of a configuration
	http.HandleFunc("/searchCPE", handleSearch(db)) // search for a CPE
	http.HandleFunc("/productVersions", handleInfo(db)) // search for a CPE

	go NotificationCron(db,2* time.Hour)
	go ImportCron(db, 24*time.Hour)

	http.ListenAndServe(":9999", nil)
}


