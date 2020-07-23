package main

import (
	"log"
	"net/http"
	"time"

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

func handleMonitor(db *sql.DB) func (http.ResponseWriter,*http.Request){
	return func(w http.ResponseWriter,r *http.Request) {
		log.Println("import")

		if r.Method != "GET" {
			http.Error(w, "Method is not supported.", http.StatusNotFound)
		}

		panic("not implemented")
	}

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
	http.HandleFunc("/monitor", handleMonitor(db))  // Send the list of cpes to be monitored (GET)
	http.HandleFunc("/searchCPE", handleSearch(db)) // search for a CPE
	go NotificationCron(db, 2*time.Hour)
	http.ListenAndServe(":9999", nil)
}


