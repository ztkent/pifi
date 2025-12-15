package networkmanager

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	connectionTimeout = 15 * time.Second
	pollInterval      = 1 * time.Second
)

func verifyAPConnection(apName string) error {
	cmd := exec.Command("nmcli", "connection", "show", apName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("AP connection not configured. Run: sudo nmcli connection add type wifi ifname wlan0 con-name PiFi-AP autoconnect no ssid PiFi mode ap 802-11-wireless.band bg")
	}
	return nil
}

func getWifiSignal() int32 {
	cmd := exec.Command("nmcli", "-f", "IN-USE,SIGNAL", "dev", "wifi", "list")
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "*") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				signal, err := strconv.ParseInt(fields[1], 10, 32)
				if err != nil {
					return -1
				}
				return int32(signal)
			}
		}
	}
	return -1
}

func getWifiMode(apName string) string {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	hasAP := strings.Contains(string(output), apName)
	hasClient := strings.Contains(string(output), "wifi") || strings.Contains(string(output), "802-11-wireless")

	if hasAP && hasClient {
		return ModeAP
	} else if hasClient {
		return ModeClient
	} else if hasAP {
		return ModeAP
	}
	return "inactive"
}

func getWifiSSID() string {
	cmd := exec.Command("nmcli", "-t", "-f", "active,ssid", "dev", "wifi")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) == 2 && fields[0] == "yes" {
			return fields[1]
		}
	}
	return ""
}

func getNetworkIps() NetworkIPs {
	status := NetworkIPs{
		WifiState: "offline",
		EthState:  "offline",
	}

	// Check WiFi
	if output, err := exec.Command("nmcli", "-g", "IP4.ADDRESS", "dev", "show", "wlan0").Output(); err == nil {
		if ip := strings.TrimSpace(string(output)); ip != "" {
			status.WifiIP = strings.Split(ip, "/")[0]
			status.WifiState = "online"
		}
	}

	// Check Ethernet
	if output, err := exec.Command("nmcli", "-g", "IP4.ADDRESS", "dev", "show", "eth0").Output(); err == nil {
		if ip := strings.TrimSpace(string(output)); ip != "" {
			status.EthernetIP = strings.Split(ip, "/")[0]
			status.EthState = "online"
		}
	}
	return status
}

func (nm *networkManager) pingTest() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-I", "wlan0", "-c", "1", "-W", "2", "1.1.1.1")
	return cmd.Run() == nil
}

func (nm *networkManager) checkWlanConnection() bool {
	cmd := exec.Command("nmcli", "-t", "-f", "DEVICE,STATE", "device")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "wlan0:connected") {
			return nm.pingTest()
		}
	}
	return false
}

func removeExistingAPs() error {
	// Get all connections
	cmd := exec.Command("nmcli", "-t", "-f", "NAME", "connection", "show")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list connections: %v", err)
	}

	// Find and delete PiFi-AP-* connections
	connections := strings.Split(string(output), "\n")
	for _, conn := range connections {
		if strings.HasPrefix(conn, "PiFi-AP-") {
			deleteCmd := exec.Command("nmcli", "connection", "delete", conn)
			if err := deleteCmd.Run(); err != nil {
				return fmt.Errorf("failed to delete connection %s: %v", conn, err)
			}
		}
	}
	return nil
}

// parseConnectionError analyzes nmcli output to determine the error type and user-friendly message
func parseConnectionError(output string) (errorType, message string) {
	outputLower := strings.ToLower(output)

	switch {
	case strings.Contains(outputLower, "secrets were required") ||
		strings.Contains(outputLower, "802-11-wireless-security") ||
		strings.Contains(outputLower, "no secrets"):
		return ErrTypeWrongPassword, "The password for this network is incorrect. Please update the password and try again."

	case strings.Contains(outputLower, "no network with ssid") ||
		strings.Contains(outputLower, "not found") ||
		strings.Contains(outputLower, "no suitable connection"):
		return ErrTypeNetworkNotFound, "The network is not available. Make sure you're in range and the network is broadcasting."

	case strings.Contains(outputLower, "timeout") ||
		strings.Contains(outputLower, "timed out"):
		return ErrTypeTimeout, "Connection timed out. The network may be too far away or experiencing issues."

	default:
		return ErrTypeUnknown, "Failed to connect to the network. Please check your settings and try again."
	}
}

// getCurrentActiveSSID returns the currently connected WiFi SSID (excluding AP connections)
func (nm *networkManager) getCurrentActiveSSID() string {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[1] == "802-11-wireless" && fields[2] == "wlan0" {
			// Skip AP connections
			if !strings.HasPrefix(fields[0], "PiFi-AP-") {
				return fields[0]
			}
		}
	}
	return ""
}

// AttemptConnectionWithFallback attempts to connect to the target network with a 15-second timeout.
// On failure, it automatically falls back to the previous network connection.
// Returns a ConnectionResult with detailed status for UI feedback.
func (nm *networkManager) AttemptConnectionWithFallback(targetSSID string) *ConnectionResult {
	nm.connectionMu.Lock()
	defer nm.connectionMu.Unlock()

	// Signal that user is attempting a connection (prevents AP watchdog interference)
	nm.userConnecting = true
	defer func() { nm.userConnecting = false }()

	result := &ConnectionResult{
		Success:      false,
		FallbackUsed: false,
	}

	// Store current connection for fallback
	previousSSID := nm.getCurrentActiveSSID()
	log.Printf("Connection attempt: %s -> %s (fallback: %s)", previousSSID, targetSSID, previousSSID)

	// Skip if already connected to target
	if previousSSID == targetSSID {
		result.Success = true
		result.NewSSID = targetSSID
		status, _ := nm.GetNetworkStatus()
		result.NewIP = status.IPs.WifiIP
		return result
	}

	// Attempt the connection
	cmd := exec.Command("nmcli", "connection", "up", targetSSID)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Connection command failed immediately
		errType, errMsg := parseConnectionError(string(output))
		result.ErrorType = errType
		result.ErrorMessage = errMsg
		log.Printf("Connection to %s failed immediately: %s", targetSSID, errMsg)

		// Attempt fallback if we had a previous connection
		if previousSSID != "" {
			result.FallbackSSID = previousSSID
			if fallbackErr := nm.restoreConnection(previousSSID); fallbackErr == nil {
				result.FallbackUsed = true
				log.Printf("Fallback to %s successful", previousSSID)
			} else {
				log.Printf("Fallback to %s failed: %v", previousSSID, fallbackErr)
			}
		}
		return result
	}

	// Connection command succeeded, now poll to verify actual connectivity
	log.Printf("Connection command accepted, polling for connectivity (timeout: %v)", connectionTimeout)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	timeout := time.After(connectionTimeout)

	for {
		select {
		case <-ticker.C:
			if nm.checkWlanConnection() {
				// Verify we're connected to the right network
				currentSSID := getWifiSSID()
				if currentSSID == targetSSID {
					result.Success = true
					result.NewSSID = targetSSID
					status, _ := nm.GetNetworkStatus()
					result.NewIP = status.IPs.WifiIP
					log.Printf("Connection to %s verified successfully (IP: %s)", targetSSID, result.NewIP)
					return result
				}
			}
		case <-timeout:
			// Timeout - connection didn't establish
			result.ErrorType = ErrTypeTimeout
			result.ErrorMessage = "Connection timed out after 15 seconds. The network may be unavailable or the password may be incorrect."
			log.Printf("Connection to %s timed out", targetSSID)

			// Attempt fallback
			if previousSSID != "" {
				result.FallbackSSID = previousSSID
				if fallbackErr := nm.restoreConnection(previousSSID); fallbackErr == nil {
					result.FallbackUsed = true
					log.Printf("Fallback to %s successful", previousSSID)
				} else {
					log.Printf("Fallback to %s failed: %v", previousSSID, fallbackErr)
				}
			}
			return result
		}
	}
}

// restoreConnection attempts to reconnect to the previous network
func (nm *networkManager) restoreConnection(ssid string) error {
	cmd := exec.Command("nmcli", "connection", "up", ssid)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restore connection to %s: %v\nOutput: %s", ssid, err, output)
	}

	// Wait briefly for connection to establish
	time.Sleep(2 * time.Second)

	if !nm.checkWlanConnection() {
		return fmt.Errorf("connection restored but connectivity check failed")
	}
	return nil
}
