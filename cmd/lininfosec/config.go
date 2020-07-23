package main

import (
	"os"
	"log"
)

var DB_DSN string
var DATA_DIR string

func LoadConfig() {
	DB_DSN = os.Getenv("LININFOSEC_MYSQL_DSN")
	DATA_DIR = os.Getenv("LININFOSEC_DATA_DIR")
	if DB_DSN == "" {
		log.Fatal("No database configured")
	}
	if DATA_DIR == "" {
		log.Fatal("No data directory configured")
	}
}
