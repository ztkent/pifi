package main

import (
	"context"
	"flag"
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

	autoAPFlag := flag.Bool("auto", true, "Enable automatic AP mode with no internet connection")
	apTimeoutFlag := flag.Int("timeout", 30, "Offline time in seconds before re-enabling AP mode")
	flag.Parse()

	r := mux.NewRouter()

	// UI routes
	r.HandleFunc("/", handlers.PiFiHandler(nm)).Methods("GET")
	r.HandleFunc("/status", handlers.StatusHandler(nm)).Methods("GET")
	r.HandleFunc("/network", handlers.NetworksHandler(nm)).Methods("GET")
	r.HandleFunc("/setmode", handlers.SetMode(nm)).Methods("POST")

	r.HandleFunc("/add-network", handlers.ModifyNetworkHandler(nm)).Methods("POST")
	r.HandleFunc("/remove-network", handlers.RemoveNetworkConnectionHandler(nm)).Methods("POST")
	r.HandleFunc("/autoconnect-network", handlers.AutoConnectNetworkHandler(nm)).Methods("POST")
	r.HandleFunc("/connect", handlers.ConnectNetworkHandler(nm)).Methods("POST")

	r.HandleFunc("/environment", handlers.EnvironmentHandler(nm)).Methods("GET", "POST")
	r.HandleFunc("/env/set", handlers.SetEnvironmentHandler(nm)).Methods("POST")
	r.HandleFunc("/env/unset", handlers.UnsetEnvironmentHandler(nm)).Methods("POST")
	r.HandleFunc("/env/set-password", handlers.SetEnvPasswordHandler(nm)).Methods("POST")
	r.HandleFunc("/env/remove-password", handlers.RemoveEnvPasswordHandler(nm)).Methods("POST")

	// API routes
	r.HandleFunc("/api/status", handlers.GetNetworkStatusAPI(nm)).Methods("GET")
	r.HandleFunc("/api/mode", handlers.SetWifiModeAPI(nm)).Methods("POST")
	r.HandleFunc("/api/networks/available", handlers.FindAvailableNetworksAPI(nm)).Methods("GET")
	r.HandleFunc("/api/networks/configured", handlers.GetConfiguredConnectionsAPI(nm)).Methods("GET")
	r.HandleFunc("/api/networks/modify", handlers.ModifyNetworkConnectionAPI(nm)).Methods("POST")
	r.HandleFunc("/api/networks/remove", handlers.RemoveNetworkConnectionAPI(nm)).Methods("DELETE")
	r.HandleFunc("/api/networks/autoconnect", handlers.SetAutoConnectConnectionAPI(nm)).Methods("POST")
	r.HandleFunc("/api/networks/connect", handlers.ConnectNetworkAPI(nm)).Methods("POST")

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8088",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if *autoAPFlag {
		go func() {
			nm.ManageOfflineAP(time.Duration(*apTimeoutFlag) * time.Second)
		}()
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
