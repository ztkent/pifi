package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/ztkent/pifi/html/handlers"
	"github.com/ztkent/pifi/networkmanager"
)

func main() {
	nm := networkmanager.New()
	err := nm.SetupAPConnection()
	if err != nil {
		log.Fatalf("Error setting up AP connection: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", handlers.StatusHandler(nm)).Methods("GET")
	r.HandleFunc("/status", handlers.StatusHandler(nm)).Methods("GET")
	r.HandleFunc("/setmode", handlers.SetMode(nm)).Methods("POST")

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		log.Printf("Server starting on http://localhost%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("PiFi Server Stopped")
}
