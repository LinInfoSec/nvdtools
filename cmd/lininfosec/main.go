package main

import (
	"log"
	"net/http"
	"os"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func handleImport (w http.ResponseWriter,r *http.Request){
	log.Println("import")

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
	}

	panic("not implemented")
}

func handleMonitor (w http.ResponseWriter,r *http.Request){
	log.Println("monitor")

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
	}

	panic("not implemented")
}

func handleSearch(w http.ResponseWriter,r *http.Request){
	log.Println("search")

	if r.Method != "POST" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
	}

	panic("not implemented")
}


func main() {
	dbTmp , err := sql.Open("mysql",os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	db = dbTmp

	http.HandleFunc("/import", handleImport)    // Triggers an import of the cpe dictionnary
	http.HandleFunc("/monitor", handleMonitor)  // Send the list of cpes to be monitored (GET)
	http.HandleFunc("/searchCPE", handleSearch) // search for a CPE
	http.ListenAndServe(":9999", nil)
}


