package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", services)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func services(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Services")
}

// COMMANDS
//
// Run
//
//---
//
// API
//
// Get Services
//
