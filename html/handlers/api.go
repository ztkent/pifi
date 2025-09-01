package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ztkent/pifi/networkmanager"
)

// API Response types
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GetNetworkStatusAPI returns the current network status as JSON
func GetNetworkStatusAPI(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status, err := nm.GetNetworkStatus()
		if err != nil {
			response := APIResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := APIResponse{
			Success: true,
			Data:    status,
		}
		json.NewEncoder(w).Encode(response)
	}
}

// SetWifiModeAPI sets the WiFi mode (client/ap) via JSON
func SetWifiModeAPI(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var request struct {
			Mode string `json:"mode"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response := APIResponse{
				Success: false,
				Error:   "Invalid JSON request body",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if request.Mode == "" {
			response := APIResponse{
				Success: false,
				Error:   "Mode parameter is required",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		err := nm.SetWifiMode(request.Mode)
		if err != nil {
			response := APIResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := APIResponse{
			Success: true,
			Data:    map[string]string{"mode": request.Mode},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// FindAvailableNetworksAPI returns available networks as JSON
func FindAvailableNetworksAPI(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		networks, err := nm.FindAvailableNetworks()
		if err != nil {
			response := APIResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := APIResponse{
			Success: true,
			Data:    map[string][]string{"networks": networks},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// GetConfiguredConnectionsAPI returns configured connections as JSON
func GetConfiguredConnectionsAPI(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		connections, err := nm.GetConfiguredConnections()
		if err != nil {
			response := APIResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := APIResponse{
			Success: true,
			Data:    map[string][]networkmanager.ConnectionInfo{"connections": connections},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// ModifyNetworkConnectionAPI creates or modifies a network connection via JSON
func ModifyNetworkConnectionAPI(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var request struct {
			SSID        string `json:"ssid"`
			Password    string `json:"password"`
			AutoConnect bool   `json:"autoConnect"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response := APIResponse{
				Success: false,
				Error:   "Invalid JSON request body",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if request.SSID == "" {
			response := APIResponse{
				Success: false,
				Error:   "SSID parameter is required",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		err := nm.ModifyNetworkConnection(request.SSID, request.Password, request.AutoConnect)
		if err != nil {
			response := APIResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := APIResponse{
			Success: true,
			Data:    map[string]string{"ssid": request.SSID, "status": "modified"},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// RemoveNetworkConnectionAPI removes a network connection via JSON
func RemoveNetworkConnectionAPI(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var request struct {
			SSID string `json:"ssid"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response := APIResponse{
				Success: false,
				Error:   "Invalid JSON request body",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if request.SSID == "" {
			response := APIResponse{
				Success: false,
				Error:   "SSID parameter is required",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		err := nm.RemoveNetworkConnection(request.SSID)
		if err != nil {
			response := APIResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := APIResponse{
			Success: true,
			Data:    map[string]string{"ssid": request.SSID, "status": "removed"},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// SetAutoConnectConnectionAPI sets auto-connect for a connection via JSON
func SetAutoConnectConnectionAPI(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var request struct {
			SSID        string `json:"ssid"`
			AutoConnect bool   `json:"autoConnect"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response := APIResponse{
				Success: false,
				Error:   "Invalid JSON request body",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if request.SSID == "" {
			response := APIResponse{
				Success: false,
				Error:   "SSID parameter is required",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		err := nm.SetAutoConnectConnection(request.SSID, request.AutoConnect)
		if err != nil {
			response := APIResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"ssid":        request.SSID,
				"autoConnect": request.AutoConnect,
				"status":      "updated",
			},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// ConnectNetworkAPI connects to a network via JSON
func ConnectNetworkAPI(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var request struct {
			SSID string `json:"ssid"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response := APIResponse{
				Success: false,
				Error:   "Invalid JSON request body",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if request.SSID == "" {
			response := APIResponse{
				Success: false,
				Error:   "SSID parameter is required",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		err := nm.ConnectNetwork(request.SSID)
		if err != nil {
			response := APIResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := APIResponse{
			Success: true,
			Data:    map[string]string{"ssid": request.SSID, "status": "connected"},
		}
		json.NewEncoder(w).Encode(response)
	}
}
