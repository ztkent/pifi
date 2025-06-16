package networkmanager

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	ModeClient   = "client"
	ModeAP       = "ap"
	passwordFile = "/etc/default/pifi_env_password"
)

type NetworkStatus struct {
	State        string
	Connectivity string
	WifiHW       string
	Wifi         string
	WifiSSID     string
	APSSID       string
	SignalStr    int32
	Mode         string
	IPs          NetworkIPs
}

type NetworkIPs struct {
	WifiIP     string
	WifiState  string
	EthernetIP string
	EthState   string
	APIP       string
	APState    string
}

type ConnectionInfo struct {
	SSID     string
	Password string
}

type NetworkManager interface {
	SetupAPConnection() error
	ManageOfflineAP(connectionLossTimeout time.Duration) error

	// Network Status
	GetNetworkStatus() (NetworkStatus, error)
	SetWifiMode(mode string) error

	// Network Configuration
	FindAvailableNetworks() ([]string, error)
	GetConfiguredConnections() ([]ConnectionInfo, error)
	ModifyNetworkConnection(ssid, password string, autoConnect bool) error
	RemoveNetworkConnection(ssid string) error
	SetAutoConnectConnection(ssid string, autoConnect bool) error
	ConnectNetwork(ssid string) error

	// Environment Management
	GetEnvironmentVariables() (map[string]string, error)
	SetEnvironmentVariable(key, value string) error
	UnsetEnvironmentVariable(key string) error
	SetEnvPassword(password string) error
	RemoveEnvPassword() error
	ValidateEnvPassword(password string) (bool, error)
	IsEnvPasswordSet() bool
}

type networkManager struct {
	status NetworkStatus
}

func New() NetworkManager {
	nm := &networkManager{
		status: NetworkStatus{
			APSSID: "PiFi-AP-" + randSeq(4),
		},
	}
	nm.GetNetworkStatus()
	return nm
}

func randSeq(n int) string {
	var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (nm *networkManager) GetNetworkStatus() (NetworkStatus, error) {
	cmd := exec.Command("nmcli", "g")
	output, err := cmd.Output()
	if err != nil {
		return nm.status, err
	}
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nm.status, fmt.Errorf("unexpected nmcli output format")
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return nm.status, fmt.Errorf("invalid nmcli output fields")
	}

	// Parse network status
	state := fields[0]
	connectivity := fields[1]
	wifiHW := fields[2]
	wifi := fields[3]
	if fields[1] == "(site" || fields[1] == "(local" && len(fields) >= 6 {
		state += " " + fields[1] + " " + fields[2]
		connectivity = fields[3]
		wifiHW = fields[4]
		wifi = fields[5]
	}

	setCase := cases.Title(language.English)
	networkStatus := NetworkStatus{
		APSSID:       nm.status.APSSID,
		State:        setCase.String(state),
		Connectivity: setCase.String(connectivity),
		WifiHW:       setCase.String(wifiHW),
		Wifi:         setCase.String(wifi),
		WifiSSID:     getWifiSSID(),
		SignalStr:    getWifiSignal(),
		Mode:         getWifiMode(nm.status.APSSID),
		IPs:          getNetworkIps(),
	}
	nm.status = networkStatus
	return networkStatus, nil
}

// Switches between client and AP modes
func (nm *networkManager) SetWifiMode(mode string) error {
	// Get current active connections
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active connections: %v", err)
	}

	hasAP := strings.Contains(string(output), nm.status.APSSID)
	hasClient := strings.Contains(string(output), "wifi") || strings.Contains(string(output), "802-11-wireless")
	switch mode {
	case ModeAP:
		if !hasClient {
			return fmt.Errorf("must have active client connection for ap mode")
		}
		if !hasAP {
			err = verifyAPConnection(nm.status.APSSID)
			if err != nil {
				return err
			}
			cmd = exec.Command("nmcli", "con", "up", nm.status.APSSID)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to create AP connection: %v\nOutput: %s", err, output)
			}
			time.Sleep(time.Second)
			newMode := getWifiMode(nm.status.APSSID)
			if newMode != "ap" {
				return fmt.Errorf("mode change verification failed")
			}
		}
	case ModeClient:
		if hasAP {
			cmd = exec.Command("nmcli", "con", "down", nm.status.APSSID)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to disable AP mode: %v", err)
			}
		}
		if !hasClient {
			return fmt.Errorf("no active client connection")
		}
		time.Sleep(time.Second)
		newMode := getWifiMode(nm.status.APSSID)
		if newMode != "inactive" && newMode != "client" {
			return fmt.Errorf("mode change verification failed")
		}
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}

	return nil
}

// Creates a new AP connection for wlan0 if it doesn't exist
func (nm *networkManager) SetupAPConnection() error {
	// Check if AP connection already exists
	cmd := exec.Command("nmcli", "connection", "show", nm.status.APSSID)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Remove all existing AP interfaces, PiFi-AP-*
	removeExistingAPs()

	// Create AP connection with required settings
	cmd = exec.Command("nmcli", "connection", "add",
		"type", "wifi",
		"ifname", "wlan0",
		"con-name", nm.status.APSSID,
		"autoconnect", "no",
		"ssid", nm.status.APSSID,
		"mode", "ap",
		"ipv4.method", "shared",
		"ipv6.method", "disabled",
		"802-11-wireless.band", "bg",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create AP connection: %v\nOutput: %s", err, output)
	}

	cmd = exec.Command("nmcli", "connection", "show", nm.status.APSSID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("AP connection verification failed: %v", err)
	}
	return nil
}

// Scan for available networks and returns a list of SSIDs
func (nm *networkManager) FindAvailableNetworks() ([]string, error) {
	// Perform a network rescan
	scanCmd := exec.Command("nmcli", "device", "wifi", "rescan")
	if err := scanCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to initiate network scan: %v", err)
	}
	time.Sleep(2 * time.Second)

	// List available networks
	cmd := exec.Command("nmcli", "--fields", "SSID", "device", "wifi", "list", "--rescan", "yes")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list available networks: %v", err)
	}

	seenNetworks := make(map[string]bool)
	networks := make([]string, 0)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		ssid := strings.TrimSpace(line)
		if ssid != "" && ssid != "SSID" && !seenNetworks[ssid] {
			seenNetworks[ssid] = true
			networks = append(networks, ssid)
		}
	}

	return networks, nil
}

// Get a list of configured connections
func (nm *networkManager) GetConfiguredConnections() ([]ConnectionInfo, error) {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "connection", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list configured connections: %v", err)
	}

	connections := make([]ConnectionInfo, 0)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) >= 2 && fields[1] == "802-11-wireless" {
			connName := fields[0]
			pskCmd := exec.Command("nmcli", "-t", "-f", "802-11-wireless-security.psk", "connection", "show", connName)
			pskOutput, _ := pskCmd.Output()
			password := strings.TrimSpace(string(pskOutput))
			connections = append(connections, ConnectionInfo{
				SSID:     connName,
				Password: password,
			})
		}
	}

	return connections, nil
}

// Modify a connection if it exists, otherwise create a new one
func (nm *networkManager) ModifyNetworkConnection(ssid, password string, autoConnect bool) error {
	checkCmd := exec.Command("nmcli", "connection", "show", ssid)
	if err := checkCmd.Run(); err == nil {
		// Connection exists - modify it
		args := []string{"connection", "modify", ssid}
		if password != "" {
			args = append(args,
				"802-11-wireless-security.key-mgmt", "wpa-psk",
				"802-11-wireless-security.psk", password)
		}
		args = append(args, "connection.autoconnect",
			map[bool]string{true: "yes", false: "no"}[autoConnect])

		cmd := exec.Command("nmcli", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to modify connection: %v\nOutput: %s", err, output)
		}
		return nil
	}

	// Connection doesn't exist - create new
	args := []string{
		"connection", "add",
		"type", "wifi",
		"ifname", "wlan0",
		"con-name", ssid,
		"autoconnect", map[bool]string{true: "yes", false: "no"}[autoConnect],
		"ssid", ssid,
	}

	if password != "" {
		args = append(args,
			"802-11-wireless-security.key-mgmt", "wpa-psk",
			"802-11-wireless-security.psk", password)
	}

	cmd := exec.Command("nmcli", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create connection: %v\nOutput: %s", err, output)
	}

	return nil
}

// Remove a saved connection by name
func (nm *networkManager) RemoveNetworkConnection(ssid string) error {
	cmd := exec.Command("nmcli", "connection", "delete", ssid)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete connection: %v", err)
	}
	return nil
}

// Set autoconnect for a saved connection by name
func (nm *networkManager) SetAutoConnectConnection(ssid string, autoConnect bool) error {
	autoConnectStr := "no"
	if autoConnect {
		autoConnectStr = "yes"
	}

	cmd := exec.Command("nmcli", "connection", "modify", ssid,
		"connection.autoconnect", autoConnectStr)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set autoconnect for %s: %v\nOutput: %s",
			ssid, err, output)
	}

	return nil
}

// Connect to a saved network by name
func (nm *networkManager) ConnectNetwork(ssid string) error {
	cmd := exec.Command("nmcli", "connection", "up", ssid)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v\nOutput: %s", ssid, err, output)
	}
	return nil
}

// Get environment variables with permission error handling
func (nm *networkManager) GetEnvironmentVariables() (map[string]string, error) {
	envVars := make(map[string]string)

	// Try to read from common environment sources
	sources := []string{
		"/etc/environment",
		"/etc/default/pifi",
		"/home/pi/.bashrc",
	}

	for _, source := range sources {
		if vars, err := readEnvFile(source); err == nil {
			for k, v := range vars {
				envVars[k] = v
			}
		}
		// Silently continue if file doesn't exist or no permission
	}

	// Add current process environment
	for _, env := range os.Environ() {
		if pair := strings.SplitN(env, "=", 2); len(pair) == 2 {
			envVars[pair[0]] = pair[1]
		}
	}

	return envVars, nil
}

// Set environment variable with graceful permission handling
func (nm *networkManager) SetEnvironmentVariable(key, value string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	// Try to write to /etc/default/pifi first
	envFile := "/etc/default/pifi"
	if err := setSystemEnv(key, value); err != nil {
		// If no permission for system file, try user file
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to set environment variable: no write access to system files and cannot determine home directory")
		}

		userEnvFile := filepath.Join(homeDir, ".pifi_env")
		if err := setSystemEnv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable: %v", err)
		}

		log.Printf("Environment variable %s set in user file %s (insufficient permissions for system file)", key, userEnvFile)
		return nil
	}

	log.Printf("Environment variable %s set in system file %s", key, envFile)
	return nil
}

// Unset environment variable with graceful permission handling
func (nm *networkManager) UnsetEnvironmentVariable(key string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	errors := []string{}

	// Try to remove from system files
	systemFiles := []string{"/etc/default/pifi", "/etc/environment"}
	for _, file := range systemFiles {
		if err := removeSystemEnv(key); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", file, err))
		}
	}

	// Try to remove from user files
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userFiles := []string{
			filepath.Join(homeDir, ".pifi_env"),
			filepath.Join(homeDir, ".bashrc"),
		}
		for _, file := range userFiles {
			if err := removeSystemEnv(key); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", file, err))
			}
		}
	}

	if len(errors) > 0 {
		log.Printf("Some errors occurred while removing environment variable %s: %s", key, strings.Join(errors, "; "))
	}

	return nil
}

// Enable the AP if there's no internet connection for a certain amount of time. This will run in the background.
func (nm *networkManager) ManageOfflineAP(connectionLossTimeout time.Duration) error {
	for {
		apMode := getWifiMode(nm.status.APSSID)
		if !nm.checkWlanConnection() && apMode != "ap" {
			log.Println("Device offline, waiting for recovery...")
			time.Sleep(connectionLossTimeout)
			if !nm.checkWlanConnection() {
				log.Println("No connection after timeout, enabling AP mode")
				if err := nm.ConnectNetwork(nm.status.APSSID); err != nil {
					log.Printf("Failed to enable AP mode: %v", err)
				}
			} else {
				log.Println("Device connection recovered")
			}
		}
		time.Sleep(60 * time.Second)
	}
}

func (nm *networkManager) SetEnvPassword(password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Hash the password
	hash := sha256.Sum256([]byte(password))
	hashedPassword := hex.EncodeToString(hash[:])

	// Try to write to system location first, fallback to user directory
	if err := writePasswordFile(passwordFile, hashedPassword); err != nil {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("failed to set password: no write access to system files and cannot determine home directory")
		}

		userPasswordFile := filepath.Join(homeDir, ".pifi_env_password")
		if err := writePasswordFile(userPasswordFile, hashedPassword); err != nil {
			return fmt.Errorf("failed to set password: %v", err)
		}
	}

	return nil
}

// RemoveEnvPassword removes the password protection
func (nm *networkManager) RemoveEnvPassword() error {
	// Try to remove from both system and user locations
	systemRemoved := os.Remove(passwordFile) == nil

	homeDir, err := os.UserHomeDir()
	userRemoved := false
	if err == nil {
		userPasswordFile := filepath.Join(homeDir, ".pifi_env_password")
		userRemoved = os.Remove(userPasswordFile) == nil
	}

	if !systemRemoved && !userRemoved {
		return fmt.Errorf("no password file found to remove")
	}

	return nil
}

// ValidateEnvPassword validates the provided password against the stored hash
func (nm *networkManager) ValidateEnvPassword(password string) (bool, error) {
	// Try system location first
	if hash, err := readPasswordFile(passwordFile); err == nil {
		return validatePassword(password, hash), nil
	}

	// Try user location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("cannot determine home directory")
	}

	userPasswordFile := filepath.Join(homeDir, ".pifi_env_password")
	if hash, err := readPasswordFile(userPasswordFile); err == nil {
		return validatePassword(password, hash), nil
	}

	return false, fmt.Errorf("no password file found")
}

// IsEnvPasswordSet checks if a password is currently set
func (nm *networkManager) IsEnvPasswordSet() bool {
	// Check system location
	if _, err := readPasswordFile(passwordFile); err == nil {
		return true
	}

	// Check user location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	userPasswordFile := filepath.Join(homeDir, ".pifi_env_password")
	_, err = readPasswordFile(userPasswordFile)
	return err == nil
}

// Helper functions
func writePasswordFile(filename, hashedPassword string) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(hashedPassword)
	return err
}

func readPasswordFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func validatePassword(password, storedHash string) bool {
	hash := sha256.Sum256([]byte(password))
	providedHash := hex.EncodeToString(hash[:])
	return providedHash == storedHash
}
