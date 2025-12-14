package handlers

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/ztkent/pifi/html"
	"github.com/ztkent/pifi/networkmanager"
)

type StatusResponse struct {
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
	Version     string    `json:"version"`
	NetworkInfo networkmanager.NetworkStatus
}

type NetworkResponse struct {
	AvailableNetworks  []string                        `json:"availableNetworks"`
	ConfiguredNetworks []networkmanager.ConnectionInfo `json:"configuredNetworks"`
	CurrentSSID        string                          `json:"currentSSID"`
	Timestamp          time.Time                       `json:"timestamp"`
}

func SetMode(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		err := nm.SetWifiMode(r.Form.Get("mode"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func StaticFileHandler() http.Handler {
	staticFS, err := fs.Sub(html.Embedded, "static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))
}

func PiFiHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFS(html.Embedded, "templates/index.gohtml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func StatusHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := StatusResponse{
			Status:    "operational",
			Timestamp: time.Now(),
			Version:   "1.0.0",
		}
		netStatus, err := nm.GetNetworkStatus()
		if err != nil {
			status.Status = fmt.Sprintf("error: %v", err)
		}
		status.NetworkInfo = netStatus

		tmpl, err := template.ParseFS(html.Embedded, "templates/status.gohtml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func NetworksHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		availableNetworks, err := nm.FindAvailableNetworks()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		configuredNetworks, err := nm.GetConfiguredConnections()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		netStatus, err := nm.GetNetworkStatus()
		var currentSSID string
		if err == nil {
			currentSSID = netStatus.WifiSSID
		}

		tmpl, err := template.ParseFS(html.Embedded, "templates/network.gohtml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		NetworkResponse := NetworkResponse{
			AvailableNetworks:  availableNetworks,
			ConfiguredNetworks: configuredNetworks,
			CurrentSSID:        currentSSID,
			Timestamp:          time.Now(),
		}
		err = tmpl.Execute(w, NetworkResponse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func ModifyNetworkHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		err := nm.ModifyNetworkConnection(r.Form.Get("ssid"), r.Form.Get("password"), false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func RemoveNetworkConnectionHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		err := nm.RemoveNetworkConnection(r.Form.Get("network"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func AutoConnectNetworkHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		err := nm.SetAutoConnectConnection(r.Form.Get("network"), true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func ConnectNetworkHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		err := nm.ConnectNetwork(r.Form.Get("network"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
