package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)




func main() {
	r := mux.NewRouter()


	srv := &http.Server{
			Handler:      r,
			Addr:         "127.0.0.1:8000",
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
	}

	log.Println("server started on port: 8080")
	log.Fatal(srv.ListenAndServe())
}