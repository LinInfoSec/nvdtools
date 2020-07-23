package main

import (
	"log"
	"net/http"
	"time"
	"encoding/json"
	"io/ioutil"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)


func handleImport(db *sql.DB) func (http.ResponseWriter,*http.Request){
	return func(w http.ResponseWriter,r *http.Request) {
		log.Println("import")

		if r.Method != "GET" {
			http.Error(w, "Method is not supported.", http.StatusNotFound)
		}

		panic("not implemented")
	}
}

func handleMonitor(db *sql.DB, action int) func (http.ResponseWriter,*http.Request){
	if action == ADD {
		return func(w http.ResponseWriter,r *http.Request) {
			log.Println("add")

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
				http.Error(w, "Bad body",http.StatusBadRequest)
				log.Println(err)
				return
			}

			if err = AddConfiguration(db, conf); err != nil {
				http.Error(w, "Could not insert",http.StatusBadRequest)
				log.Println(err)
				return
			}
		}
	} else if action == UPDATE {
		return func(w http.ResponseWriter,r *http.Request) {
			log.Println("update")

			if r.Method != "POST" {
				http.Error(w, "Method is not supported.", http.StatusNotFound)
			}

			panic("not implemented")
		}
	} else if action == REMOVE {
		return func(w http.ResponseWriter,r *http.Request) {
			log.Println("remove")

			if r.Method != "POST" {
				http.Error(w, "Method is not supported.", http.StatusNotFound)
			}

			panic("not implemented")
		}
	}

	panic("unknown action")

}

func handleSearch(db *sql.DB) func (http.ResponseWriter,*http.Request){
	return func(w http.ResponseWriter,r *http.Request) {
		log.Println("import")

		if r.Method != "GET" {
			http.Error(w, "Method is not supported.", http.StatusNotFound)
		}

		panic("not implemented")
	}

}




func main() {
	LoadConfig()

	db, err := sql.Open("mysql",DB_DSN)
	if err != nil {
		log.Fatal(err)
	}


	http.HandleFunc("/import", handleImport(db))    // Triggers an import of the cpe dictionnary
	http.HandleFunc("/monitor/add", handleMonitor(db,ADD))  // Add configurations to be monitored
	http.HandleFunc("/monitor/remove", handleMonitor(db,REMOVE))  // Remove configurations to be monitored
	http.HandleFunc("/monitor/update", handleMonitor(db,UPDATE))  // Remove configurations to be monitored
	http.HandleFunc("/searchCPE", handleSearch(db)) // search for a CPE
	go NotificationCron(db, 2*time.Hour)
	http.ListenAndServe(":9999", nil)
}


