package main

import (
	"os"
	"log"
)

var DB_DSN string
var DATA_DIR string
var NOTIFICATION_ENDPOINT string

func LoadConfig() {
	DB_DSN = os.Getenv("LININFOSEC_MYSQL_DSN")
	DATA_DIR = os.Getenv("LININFOSEC_DATA_DIR")
	NOTIFICATION_ENDPOINT = os.Getenv("LININFOSEC_NOTIFICATION_ENDPOINT")
	if DB_DSN == "" {
		log.Fatal("No database configured")
	}
	if DATA_DIR == "" {
		log.Fatal("No data directory configured")
	}
}
