package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lupppig/forge-vod/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	_ = cfg

	r := mux.NewRouter()


	srv := &http.Server{
			Handler:      r,
			Addr:         ":8080",
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
	}

	log.Println("server started on port: 8080")
	log.Fatal(srv.ListenAndServe())
}