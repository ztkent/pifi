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
	r.HandleFunc("/", handlers.PiFiHandler(nm)).Methods("GET")
	r.HandleFunc("/status", handlers.StatusHandler(nm)).Methods("GET")
	r.HandleFunc("/network", handlers.NetworksHandler(nm)).Methods("GET")
	r.HandleFunc("/setmode", handlers.SetMode(nm)).Methods("POST")

	//r.HandleFunc("/add-network", handlers.ModifyNetworkHandler(nm)).Methods("POST")
	// r.HandleFunc("/remove-network", handlers.RemoveNetworkConnectionHandler(nm)).Methods("POST")
	// r.HandleFunc("/autoconnect-network", handlers.AutoConnectNetworkHandler(nm)).Methods("POST")
	// r.HandleFunc("/connect", handlers.ConnectNetworkHandler(nm)).Methods("POST")

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8088",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		log.Printf("Server starting on http://%s", srv.Addr)
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
