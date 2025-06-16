package handlers

import (
	"fmt"
	"html/template"
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
	Timestamp          time.Time                       `json:"timestamp"`
}

type EnvironmentResponse struct {
	EnvironmentVars map[string]string `json:"environmentVars"`
	Timestamp       time.Time         `json:"timestamp"`
	IsPasswordSet   bool              `json:"isPasswordSet"`
	RequiresAuth    bool              `json:"requiresAuth"`
}

type PasswordResponse struct {
	IsPasswordSet bool `json:"isPasswordSet"`
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

func PiFiHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFS(html.Templates, "templates/index.gohtml")
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

		tmpl, err := template.ParseFS(html.Templates, "templates/status.gohtml")
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

		tmpl, err := template.ParseFS(html.Templates, "templates/network.gohtml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		NetworkResponse := NetworkResponse{
			AvailableNetworks:  availableNetworks,
			ConfiguredNetworks: configuredNetworks,
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

func EnvironmentHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isPasswordSet := nm.IsEnvPasswordSet()

		// Check if password is required and validate session/password
		if isPasswordSet {
			// Check if password was provided
			r.ParseForm()
			password := r.Form.Get("password")

			if password == "" {
				// Show password prompt
				response := EnvironmentResponse{
					EnvironmentVars: nil,
					Timestamp:       time.Now(),
					IsPasswordSet:   true,
					RequiresAuth:    true,
				}

				tmpl, err := template.ParseFS(html.Templates, "templates/envs.gohtml")
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				err = tmpl.Execute(w, response)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				return
			}

			// Validate password
			valid, err := nm.ValidateEnvPassword(password)
			if err != nil || !valid {
				http.Error(w, "Invalid password", http.StatusUnauthorized)
				return
			}
		}

		// Password validated or not required, show environment variables
		envVars, err := nm.GetEnvironmentVariables()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := EnvironmentResponse{
			EnvironmentVars: envVars,
			Timestamp:       time.Now(),
			IsPasswordSet:   isPasswordSet,
			RequiresAuth:    false,
		}

		tmpl, err := template.ParseFS(html.Templates, "templates/envs.gohtml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func SetEnvironmentHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check password protection
		if nm.IsEnvPasswordSet() {
			r.ParseForm()
			password := r.Form.Get("auth_password")
			if password == "" {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			valid, err := nm.ValidateEnvPassword(password)
			if err != nil || !valid {
				http.Error(w, "Invalid password", http.StatusUnauthorized)
				return
			}
		}

		r.ParseForm()
		key := r.Form.Get("key")
		value := r.Form.Get("value")

		if key == "" {
			http.Error(w, "Environment variable key cannot be empty", http.StatusBadRequest)
			return
		}

		err := nm.SetEnvironmentVariable(key, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func UnsetEnvironmentHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check password protection
		if nm.IsEnvPasswordSet() {
			r.ParseForm()
			password := r.Form.Get("auth_password")
			if password == "" {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			valid, err := nm.ValidateEnvPassword(password)
			if err != nil || !valid {
				http.Error(w, "Invalid password", http.StatusUnauthorized)
				return
			}
		}

		r.ParseForm()
		key := r.Form.Get("key")

		if key == "" {
			http.Error(w, "Environment variable key cannot be empty", http.StatusBadRequest)
			return
		}

		err := nm.UnsetEnvironmentVariable(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func SetEnvPasswordHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		password := r.Form.Get("new_password")
		confirmPassword := r.Form.Get("confirm_password")

		if password == "" {
			http.Error(w, "Password cannot be empty", http.StatusBadRequest)
			return
		}

		if password != confirmPassword {
			http.Error(w, "Passwords do not match", http.StatusBadRequest)
			return
		}

		err := nm.SetEnvPassword(password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func RemoveEnvPasswordHandler(nm networkmanager.NetworkManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify current password before removing
		r.ParseForm()
		currentPassword := r.Form.Get("current_password")

		if currentPassword == "" {
			http.Error(w, "Current password required", http.StatusBadRequest)
			return
		}

		valid, err := nm.ValidateEnvPassword(currentPassword)
		if err != nil || !valid {
			http.Error(w, "Invalid current password", http.StatusUnauthorized)
			return
		}

		err = nm.RemoveEnvPassword()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
